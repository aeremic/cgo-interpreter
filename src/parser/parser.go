package parser

import (
	"fmt"
	"strconv"

	"github.com/aeremic/cgo/ast"
	"github.com/aeremic/cgo/token"
	"github.com/aeremic/cgo/tokenizer"
)

type Parser struct {
	tokenizer      *tokenizer.Tokenizer // Pointer to the lexer
	currentToken   token.Token
	peekToken      token.Token
	prefixParseFns map[token.Type]prefixParseFn
	infixParseFns  map[token.Type]infixParseFn

	errors []string
}

// New Constructor
func New(t *tokenizer.Tokenizer) *Parser {
	p := &Parser{tokenizer: t, errors: []string{}}

	// Call nextToken two times to initialize
	// both current token and next token
	p.nextToken()
	p.nextToken()

	p.prefixParseFns = make(map[token.Type]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNC, p.parseFunctionLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseDictLiteral)

	p.infixParseFns = make(map[token.Type]infixParseFn)
	p.registerInfix(token.EQUALS, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQUALS, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)

	return p
}

// Methods

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) LogPeekError(t token.Type) {
	msg := fmt.Sprintf("Expected %s token. Got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) ParseProgram() *ast.ProgramRoot {
	program := &ast.ProgramRoot{}
	program.Statements = []ast.Statement{}

	for !p.checkCurrentTokenType(token.EOF) {
		statement := p.parseStatement()
		if statement != nil {
			program.Statements = append(program.Statements, statement)
		}

		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.currentToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	statement := &ast.LetStatement{Token: p.currentToken}

	if !p.peekAndMove(token.IDENT) {
		return nil
	}

	statement.Name = &ast.Identifier{Token: p.currentToken, Value: p.currentToken.Literal}

	if !p.peekAndMove(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	statement.Value = p.parseExpression(LOWEST)

	for !p.checkCurrentTokenType(token.SEMICOLON) {
		p.nextToken()
	}

	return statement
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	statement := &ast.ReturnStatement{Token: p.currentToken}

	p.nextToken()

	statement.ReturnValue = p.parseExpression(LOWEST)

	for !p.checkCurrentTokenType(token.SEMICOLON) {
		p.nextToken()
	}

	return statement
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	statement := &ast.ExpressionStatement{Token: p.currentToken}

	statement.Expression = p.parseExpression(LOWEST)

	if p.checkPeekTokenType(token.SEMICOLON) {
		p.nextToken()
	}

	return statement
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.currentToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.currentToken.Type)
		return nil
	}

	leftExpression := prefix()

	for !p.checkPeekTokenType(token.SEMICOLON) && precedence < p.peekTokenPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExpression
		}

		p.nextToken()

		leftExpression = infix(leftExpression)
	}

	return leftExpression
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.currentToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.checkCurrentTokenType(token.RBRACE) && !p.checkCurrentTokenType(token.EOF) {
		statement := p.parseStatement()
		if statement != nil {
			block.Statements = append(block.Statements, statement)
		}

		p.nextToken()
	}

	return block
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.currentToken, Value: p.currentToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	literal := &ast.IntegerLiteral{Token: p.currentToken}

	value, err := strconv.ParseInt(p.currentToken.Literal, 0, 64)
	if err != nil {
		message := fmt.Sprintf("Could not parse %q as integer.", p.currentToken.Literal)
		p.errors = append(p.errors, message)

		return nil
	}

	literal.Value = value

	return literal
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{
		Token: p.currentToken,
		Value: p.currentToken.Literal,
	}
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expression := &ast.PrefixExpression{
		Token:    p.currentToken,
		Operator: p.currentToken.Literal,
	}

	p.nextToken()

	expression.Right = p.parseExpression(PREFIX)

	return expression
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expression := &ast.InfixExpression{
		Token:    p.currentToken,
		Operator: p.currentToken.Literal,
		Left:     left,
	}

	precedence := p.currentTokenPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	expression := &ast.CallExpression{
		Token:    p.currentToken,
		Function: function,
	}

	expression.Arguments = p.parseExpressionList(token.RPAREN)

	return expression
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	expression := &ast.IndexExpression{
		Token: p.currentToken,
		Left:  left,
	}

	p.nextToken()
	expression.Index = p.parseExpression(LOWEST)

	if !p.peekAndMove(token.RBRACKET) {
		return nil
	}

	return expression
}

func (p *Parser) parseCallArguments() []ast.Expression {
	args := []ast.Expression{}

	if p.checkPeekTokenType(token.RPAREN) {
		p.nextToken()
		return args
	}

	p.nextToken()

	args = append(args, p.parseExpression(LOWEST))
	for p.checkPeekTokenType(token.COMMA) {
		p.nextToken()
		p.nextToken()

		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.peekAndMove(token.RPAREN) {
		return nil
	}

	return args
}

func (p *Parser) parseBoolean() ast.Expression {
	expression := &ast.Boolean{
		Token: p.currentToken,
		Value: p.checkCurrentTokenType(token.TRUE),
	}

	return expression
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	expression := p.parseExpression(LOWEST)

	if !p.checkPeekTokenType(token.RPAREN) {
		return nil
	}

	p.nextToken()

	return expression
}

func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{
		Token: p.currentToken,
	}

	if !p.checkPeekTokenType(token.LPAREN) {
		return nil
	}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if !p.checkCurrentTokenType(token.RPAREN) {
		return nil
	}

	if !p.checkPeekTokenType(token.LBRACE) {
		return nil
	}

	p.nextToken()
	expression.Consequence = p.parseBlockStatement()

	if p.checkPeekTokenType(token.ELSE) {
		p.nextToken()

		if !p.checkPeekTokenType(token.LBRACE) {
			return nil
		}

		p.nextToken()
		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	literal := &ast.FunctionLiteral{
		Token: p.currentToken,
	}

	if !p.peekAndMove(token.LPAREN) {
		return nil
	}

	literal.Parameters = p.parseFunctionParameters()

	if !p.peekAndMove(token.LBRACE) {
		return nil
	}

	literal.Body = p.parseBlockStatement()

	return literal
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := []*ast.Identifier{}

	if p.checkPeekTokenType(token.RPAREN) {
		p.nextToken()

		return identifiers
	}

	p.nextToken()

	identifiers = append(identifiers, &ast.Identifier{
		Token: p.currentToken,
		Value: p.currentToken.Literal,
	})

	for p.checkPeekTokenType(token.COMMA) {
		p.nextToken()
		p.nextToken()

		identifiers = append(identifiers, &ast.Identifier{
			Token: p.currentToken,
			Value: p.currentToken.Literal,
		})
	}

	if !p.peekAndMove(token.RPAREN) {
		return nil
	}

	return identifiers
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{
		Token: p.currentToken,
	}

	array.Elements = p.parseExpressionList(token.RBRACKET)

	return array
}

func (p *Parser) parseDictLiteral() ast.Expression {
	dict := &ast.DictLiteral{Token: p.currentToken}
	dict.Elements = make(map[ast.Expression]ast.Expression)

	for !p.checkPeekTokenType(token.RBRACE) {
		p.nextToken()
		key := p.parseExpression(LOWEST)

		if !p.peekAndMove(token.COLON) {
			return nil
		}

		p.nextToken()
		value := p.parseExpression(LOWEST)

		dict.Elements[key] = value

		if !p.checkPeekTokenType(token.RBRACE) && !p.peekAndMove(token.COMMA) {
			return nil
		}
	}

	if !p.peekAndMove(token.RBRACE) {
		return nil
	}

	return dict
}

func (p *Parser) parseExpressionList(end token.Type) []ast.Expression {
	list := []ast.Expression{}

	if p.checkPeekTokenType(end) {
		p.nextToken()
		return list
	}

	p.nextToken()

	list = append(list, p.parseExpression(LOWEST))
	for p.checkPeekTokenType(token.COMMA) {
		p.nextToken()
		p.nextToken()

		list = append(list, p.parseExpression(LOWEST))
	}

	if !p.peekAndMove(end) {
		return nil
	}

	return list
}
