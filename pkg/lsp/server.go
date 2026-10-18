package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/Nehonix-Team/xru/internal/utils"
	"github.com/Nehonix-Team/xru/pkg/formatter"
	"github.com/Nehonix-Team/xru/pkg/lsp/navigation"
)

type Server struct {
	reader    *bufio.Reader
	writer    io.Writer
	mu        sync.Mutex
	docs      map[string]string // URI -> Content
	navigator *navigation.Navigator
}

func NewServer(in io.Reader, out io.Writer) *Server {
	return &Server{
		reader: bufio.NewReader(in),
		writer: out,
		docs:   make(map[string]string),
	}
}

func (s *Server) Run() error {
	for {
		req, err := s.readMessage()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if err := s.handleRequest(req); err != nil {
			// Continue on request errors
		}
	}
}

func (s *Server) readMessage() (*Request, error) {
	contentLength := 0
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				cl, err := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err == nil {
					contentLength = cl
				}
			}
		}
	}

	if contentLength == 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	body := make([]byte, contentLength)
	_, err := io.ReadFull(s.reader, body)
	if err != nil {
		return nil, err
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}

	return &req, nil
}

func (s *Server) writeResponse(id any, result any) error {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return s.send(resp)
}

func (s *Server) writeNotification(method string, params any) error {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	return s.send(notif)
}

func (s *Server) send(payload any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := s.writer.Write([]byte(header)); err != nil {
		return err
	}
	_, err = s.writer.Write(data)
	return err
}

func (s *Server) handleRequest(req *Request) error {
	switch req.Method {
	case "initialize":
		var params InitializeParams
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params, &params)
		}
		res := InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync: TextDocumentSyncOptions{
					OpenClose: true,
					Change:    1, // Full text sync
				},
				HoverProvider:             true,
				DefinitionProvider:        true,
				DocumentSymbolProvider:    true,
				CompletionProvider:         &CompletionOptions{TriggerCharacters: []string{".", "@", ":", " ", "\"", "["}},
				DocumentHighlightProvider:  true,
				ReferencesProvider:         true,
				DocumentFormattingProvider: true,
			},
			ServerInfo: &ServerInfo{
				Name:    "xru-lsp",
				Version: utils.BinVersion,
			},
		}
		return s.writeResponse(req.ID, res)

	case "initialized":
		return nil

	case "textDocument/didOpen":
		var params DidOpenTextDocumentParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		s.docs[params.TextDocument.URI] = params.TextDocument.Text
		s.validateAndPublish(params.TextDocument.URI, params.TextDocument.Text)
		return nil

	case "textDocument/didChange":
		var params DidChangeTextDocumentParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		if len(params.ContentChanges) > 0 {
			latest := params.ContentChanges[len(params.ContentChanges)-1].Text
			s.docs[params.TextDocument.URI] = latest
			s.validateAndPublish(params.TextDocument.URI, latest)
		}
		return nil

	case "textDocument/didClose":
		var params DidCloseTextDocumentParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		delete(s.docs, params.TextDocument.URI)
		_ = s.writeNotification("textDocument/publishDiagnostics", PublishDiagnosticsParams{
			URI:         params.TextDocument.URI,
			Diagnostics: []Diagnostic{},
		})
		return nil

	case "textDocument/hover":
		var params HoverParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		hover := s.handleHover(params.TextDocument.URI, params.Position)
		return s.writeResponse(req.ID, hover)

	case "textDocument/definition":
		var params DefinitionParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		loc := s.handleDefinition(params.TextDocument.URI, params.Position)
		return s.writeResponse(req.ID, loc)

	case "textDocument/documentSymbol":
		var params DocumentSymbolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		symbols := s.handleDocumentSymbols(params.TextDocument.URI)
		return s.writeResponse(req.ID, symbols)

	case "textDocument/completion":
		var params CompletionParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		completions := s.handleCompletion(params.TextDocument.URI, params.Position)
		return s.writeResponse(req.ID, completions)

	case "textDocument/documentHighlight":
		var params DocumentHighlightParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		highlights := s.handleDocumentHighlight(params.TextDocument.URI, params.Position)
		return s.writeResponse(req.ID, highlights)

	case "textDocument/references":
		var params ReferenceParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		refs := s.handleReferences(params.TextDocument.URI, params.Position)
		return s.writeResponse(req.ID, refs)

	case "textDocument/codeAction":
		var params CodeActionParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		actions := s.handleCodeAction(params.TextDocument.URI, params.Range, params.Context)
		return s.writeResponse(req.ID, actions)

	case "textDocument/formatting":
		var params DocumentFormattingParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return err
		}
		edits := s.handleFormatting(params.TextDocument.URI)
		return s.writeResponse(req.ID, edits)

	case "shutdown":
		return s.writeResponse(req.ID, nil)

	case "exit":
		os.Exit(0)
		return nil
	}

	return nil
}

func (s *Server) handleFormatting(uri string) []TextEdit {
	content, ok := s.docs[uri]
	if !ok {
		return []TextEdit{}
	}
	formatted, err := formatter.Format(content)
	if err != nil || formatted == content {
		return []TextEdit{}
	}

	lines := strings.Split(content, "\n")
	lastLine := len(lines) - 1
	lastChar := len(lines[lastLine])

	return []TextEdit{
		{
			Range: Range{
				Start: Position{Line: 0, Character: 0},
				End:   Position{Line: lastLine, Character: lastChar},
			},
			NewText: formatted,
		},
	}
}
