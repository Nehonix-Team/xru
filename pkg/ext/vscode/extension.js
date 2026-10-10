const vscode = require("vscode");
const path = require("path");
const fs = require("fs");
const { XruLspClient } = require("./lsp_client");

let diagnosticCollection;
let lspClient = null;
let statusBarItem = null;

function toVsCodeRange(range) {
  return new vscode.Range(
    new vscode.Position(range.start.line, range.start.character),
    new vscode.Position(range.end.line, range.end.character),
  );
}

function findLspBinary() {
  // 1. Configured custom path in VS Code settings
  const configured = vscode.workspace.getConfiguration("xru").get("lspPath");
  if (configured && fs.existsSync(configured)) {
    return configured;
  }

  const binName = process.platform === "win32" ? "xru.exe" : "xru";

  // 2. Search workspace folders dynamically (mods/xru, lib/xru, bin, root)
  if (vscode.workspace.workspaceFolders) {
    for (const wf of vscode.workspace.workspaceFolders) {
      const candidates = [
        path.join(wf.uri.fsPath, "mods", "xru", binName),
        path.join(wf.uri.fsPath, "lib", "xru", binName),
        path.join(wf.uri.fsPath, "bin", binName),
        path.join(wf.uri.fsPath, binName),
      ];
      for (const cand of candidates) {
        if (fs.existsSync(cand)) {
          return cand;
        }
      }
    }
  }

  // 3. Search in system PATH
  const envPath = process.env.PATH || "";
  const pathDirs = envPath.split(path.delimiter);
  for (const dir of pathDirs) {
    if (!dir) continue;
    const candidate = path.join(dir, binName);
    if (fs.existsSync(candidate)) {
      try {
        if (process.platform !== "win32") {
          fs.accessSync(candidate, fs.constants.X_OK);
        }
        return candidate;
      } catch (e) {
        // Ignore if not executable
      }
    }
  }

  return null;
}

function syncDocumentToLsp(document, isNew = false) {
  if (
    !document ||
    (document.languageId !== "xru" && !document.fileName.endsWith(".xru"))
  ) {
    return;
  }
  if (!lspClient || !lspClient.isAlive()) {
    startLspServer().then((ok) => {
      if (ok && lspClient && lspClient.isAlive()) {
        syncDocumentToLsp(document, true);
      }
    });
    return;
  }

  const uriStr = document.uri.toString();
  const text = document.getText();

  if (isNew) {
    lspClient.notify("textDocument/didOpen", {
      textDocument: {
        uri: uriStr,
        languageId: "xru",
        version: document.version,
        text: text,
      },
    });
  } else {
    lspClient.notify("textDocument/didChange", {
      textDocument: {
        uri: uriStr,
        version: document.version,
      },
      contentChanges: [{ text: text }],
    });
  }
}

let isStartingLsp = false;

async function startLspServer() {
  if (lspClient && lspClient.isAlive()) {
    return true;
  }
  if (isStartingLsp) {
    return false;
  }
  isStartingLsp = true;

  const binPath = findLspBinary();
  if (!binPath) {
    updateStatusBar(false, "XRU: LSP not found (click to retry)");
    isStartingLsp = false;
    return false;
  }

  try {
    const client = new XruLspClient(binPath);
    client.onDiagnostics((params) => {
      const targetUri = vscode.Uri.parse(params.uri);
      const items = params.diagnostics || [];
      if (items.length === 0) {
        diagnosticCollection.delete(targetUri);
        diagnosticCollection.set(targetUri, []);
        return;
      }
      const diags = items.map((d) => {
        const range = toVsCodeRange(d.range);
        const diag = new vscode.Diagnostic(
          range,
          d.message,
          d.severity === 1
            ? vscode.DiagnosticSeverity.Error
            : vscode.DiagnosticSeverity.Warning,
        );
        diag.source = d.source || "xru";
        return diag;
      });
      diagnosticCollection.set(targetUri, diags);
    });

    client.onExit((code) => {
      lspClient = null;
      updateStatusBar(false, "XRU: Server stopped (click to restart)");
      setTimeout(() => {
        startLspServer();
      }, 1500);
    });

    const initRes = await client.start();
    lspClient = client;
    isStartingLsp = false;

    const serverVer =
      initRes && initRes.serverInfo ? initRes.serverInfo.version : "2.0";
    updateStatusBar(true, `XRU (${serverVer})`);

    // Send all currently opened .xru documents
    vscode.workspace.textDocuments.forEach((doc) => {
      syncDocumentToLsp(doc, true);
    });

    return true;
  } catch (e) {
    lspClient = null;
    isStartingLsp = false;
    updateStatusBar(false, `XRU LSP Error (click to retry)`);
    return false;
  }
}

function updateStatusBar(online, text) {
  if (!statusBarItem) return;
  statusBarItem.text = `$(gear) ${text}`;
  statusBarItem.tooltip = online
    ? "XRU 2.0 Language Server is running"
    : "Configure xru.lspPath or add xru to PATH";
  statusBarItem.show();
}

function activate(context) {
  diagnosticCollection = vscode.languages.createDiagnosticCollection("xru");
  context.subscriptions.push(diagnosticCollection);

  statusBarItem = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Right,
    100,
  );
  statusBarItem.command = "xru.restartServer";
  context.subscriptions.push(statusBarItem);

  context.subscriptions.push(
    vscode.commands.registerCommand("xru.restartServer", async () => {
      if (lspClient) {
        lspClient.stop();
        lspClient = null;
      }
      diagnosticCollection.clear();
      const ok = await startLspServer();
      if (ok) {
        vscode.window.showInformationMessage("XRU Language Server restarted");
      }
    }),
  );

  // 1. Start Native Go LSP Server
  startLspServer();

  // 2. Document lifecycle synchronization
  context.subscriptions.push(
    vscode.workspace.onDidOpenTextDocument((doc) => {
      syncDocumentToLsp(doc, true);
    }),
  );

  let changeDebounce = null;
  context.subscriptions.push(
    vscode.workspace.onDidChangeTextDocument((event) => {
      if (changeDebounce) {
        clearTimeout(changeDebounce);
      }
      changeDebounce = setTimeout(() => {
        syncDocumentToLsp(event.document, false);
      }, 75);
    }),
  );

  context.subscriptions.push(
    vscode.workspace.onDidSaveTextDocument((doc) => {
      syncDocumentToLsp(doc, false);
    }),
  );

  context.subscriptions.push(
    vscode.workspace.onDidCloseTextDocument((doc) => {
      diagnosticCollection.delete(doc.uri);
      if (lspClient && lspClient.isAlive()) {
        lspClient.notify("textDocument/didClose", {
          textDocument: { uri: doc.uri.toString() },
        });
      }
    }),
  );

  // 3. Definition Provider (Ctrl+Click / F12)
  const definitionProvider = vscode.languages.registerDefinitionProvider(
    "xru",
    {
      async provideDefinition(document, position, token) {
        if (!lspClient || !lspClient.isAlive()) return null;
        try {
          if (changeDebounce) {
            clearTimeout(changeDebounce);
            changeDebounce = null;
            syncDocumentToLsp(document, false);
          }
          const res = await lspClient.request("textDocument/definition", {
            textDocument: { uri: document.uri.toString() },
            position: { line: position.line, character: position.character },
          });
          if (res && res.uri && res.range) {
            return new vscode.Location(
              vscode.Uri.parse(res.uri),
              toVsCodeRange(res.range),
            );
          }
        } catch (e) {}
        return null;
      },
    },
  );
  context.subscriptions.push(definitionProvider);

  // 4. Hover Provider
  const hoverProvider = vscode.languages.registerHoverProvider("xru", {
    async provideHover(document, position, token) {
      if (!lspClient || !lspClient.isAlive()) return null;
      try {
        if (changeDebounce) {
          clearTimeout(changeDebounce);
          changeDebounce = null;
          syncDocumentToLsp(document, false);
        }
        const res = await lspClient.request("textDocument/hover", {
          textDocument: { uri: document.uri.toString() },
          position: { line: position.line, character: position.character },
        });
        if (res && res.contents && res.contents.value) {
          const md = new vscode.MarkdownString(res.contents.value);
          md.isTrusted = true;
          return new vscode.Hover(md);
        }
      } catch (e) {}
      return null;
    },
  });
  context.subscriptions.push(hoverProvider);

  // 5. Completion Item Provider
  const completionProvider = vscode.languages.registerCompletionItemProvider(
    "xru",
    {
      async provideCompletionItems(document, position, token, context) {
        if (!lspClient || !lspClient.isAlive()) return null;
        try {
          if (changeDebounce) {
            clearTimeout(changeDebounce);
            changeDebounce = null;
          }
          syncDocumentToLsp(document, false);
          const res = await lspClient.request("textDocument/completion", {
            textDocument: { uri: document.uri.toString() },
            position: { line: position.line, character: position.character },
          });
          if (res && Array.isArray(res.items)) {
            const items = res.items.map((it) => {
              const item = new vscode.CompletionItem(it.label);
              item.detail = it.detail;
              if (it.documentation) {
                item.documentation = new vscode.MarkdownString(
                  typeof it.documentation === "string"
                    ? it.documentation
                    : it.documentation.value,
                );
              }
              if (it.insertTextFormat === 2) {
                item.insertText = new vscode.SnippetString(
                  it.insertText || it.label,
                );
              } else if (it.insertText) {
                item.insertText = it.insertText;
              }
              if (
                it.additionalTextEdits &&
                Array.isArray(it.additionalTextEdits)
              ) {
                item.additionalTextEdits = it.additionalTextEdits.map((te) => {
                  return new vscode.TextEdit(
                    toVsCodeRange(te.range),
                    te.newText,
                  );
                });
              }
              return item;
            });
            return new vscode.CompletionList(items, true);
          }
        } catch (e) {}
        return null;
      },
    },
    ".",
    "@",
    ":",
    " ",
    "\"",
  );
  context.subscriptions.push(completionProvider);

  // 6. Code Action Provider (Quick Fix / Auto-Import on Lightbulb & Ctrl+.)
  const codeActionProvider = vscode.languages.registerCodeActionsProvider(
    "xru",
    {
      async provideCodeActions(document, range, context, token) {
        if (!lspClient || !lspClient.isAlive()) return [];
        try {
          const res = await lspClient.request("textDocument/codeAction", {
            textDocument: { uri: document.uri.toString() },
            range: {
              start: {
                line: range.start.line,
                character: range.start.character,
              },
              end: { line: range.end.line, character: range.end.character },
            },
            context: {
              diagnostics: (context.diagnostics || []).map((d) => ({
                range: {
                  start: {
                    line: d.range.start.line,
                    character: d.range.start.character,
                  },
                  end: {
                    line: d.range.end.line,
                    character: d.range.end.character,
                  },
                },
                severity:
                  d.severity === vscode.DiagnosticSeverity.Error ? 1 : 2,
                source: d.source || "xru",
                message: d.message,
              })),
            },
          });

          if (Array.isArray(res)) {
            return res.map((act) => {
              const ca = new vscode.CodeAction(
                act.title,
                vscode.CodeActionKind.QuickFix,
              );
              ca.isPreferred = act.isPreferred || false;
              if (act.edit && act.edit.changes) {
                const we = new vscode.WorkspaceEdit();
                for (const [uriStr, edits] of Object.entries(
                  act.edit.changes,
                )) {
                  const targetUri = vscode.Uri.parse(uriStr);
                  for (const e of edits) {
                    we.replace(targetUri, toVsCodeRange(e.range), e.newText);
                  }
                }
                ca.edit = we;
              }
              return ca;
            });
          }
        } catch (e) {}
        return [];
      },
    },
    {
      providedCodeActionKinds: [vscode.CodeActionKind.QuickFix],
    },
  );
  context.subscriptions.push(codeActionProvider);

  // 7. Document Highlight Provider (Highlight occurrences of word at cursor)
  const highlightProvider = vscode.languages.registerDocumentHighlightProvider(
    "xru",
    {
      async provideDocumentHighlights(document, position, token) {
        if (!lspClient || !lspClient.isAlive()) return [];
        try {
          const res = await lspClient.request(
            "textDocument/documentHighlight",
            {
              textDocument: { uri: document.uri.toString() },
              position: { line: position.line, character: position.character },
            },
          );
          if (Array.isArray(res)) {
            return res.map(
              (h) =>
                new vscode.DocumentHighlight(
                  toVsCodeRange(h.range),
                  vscode.DocumentHighlightKind.Read,
                ),
            );
          }
        } catch (e) {}
        return [];
      },
    },
  );
  context.subscriptions.push(highlightProvider);

  // 8. Reference Provider (Find All References)
  const referenceProvider = vscode.languages.registerReferenceProvider("xru", {
    async provideReferences(document, position, context, token) {
      if (!lspClient || !lspClient.isAlive()) return [];
      try {
        const res = await lspClient.request("textDocument/references", {
          textDocument: { uri: document.uri.toString() },
          position: { line: position.line, character: position.character },
          context: { includeDeclaration: true },
        });
        if (Array.isArray(res)) {
          return res.map(
            (r) =>
              new vscode.Location(
                vscode.Uri.parse(r.uri),
                toVsCodeRange(r.range),
              ),
          );
        }
      } catch (e) {}
      return [];
    },
  });
  context.subscriptions.push(referenceProvider);

  // 9. Document Symbol Provider (Outline & Breadcrumbs)
  const symbolProvider = vscode.languages.registerDocumentSymbolProvider(
    "xru",
    {
      async provideDocumentSymbols(document, token) {
        if (!lspClient || !lspClient.isAlive()) return [];
        try {
          const res = await lspClient.request("textDocument/documentSymbol", {
            textDocument: { uri: document.uri.toString() },
          });
          if (Array.isArray(res) && res.length > 0) {
            return res.map(
              (s) =>
                new vscode.DocumentSymbol(
                  s.name,
                  s.detail || "",
                  s.kind
                    ? s.kind === 2
                      ? vscode.SymbolKind.Module
                      : s.kind === 5
                        ? vscode.SymbolKind.Class
                        : s.kind === 12
                          ? vscode.SymbolKind.Function
                          : vscode.SymbolKind.Variable
                    : vscode.SymbolKind.Variable,
                  toVsCodeRange(s.range),
                  toVsCodeRange(s.range),
                ),
            );
          }
        } catch (e) {}
        return [];
      },
    },
  );
  context.subscriptions.push(symbolProvider);
}

function deactivate() {
  if (lspClient) {
    lspClient.stop();
    lspClient = null;
  }
}

module.exports = {
  activate,
  deactivate,
};
