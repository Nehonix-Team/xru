const vscode = require('vscode');
const path = require('path');
const { WorkspaceIndex } = require('./analyzer');

const workspaceIndex = new WorkspaceIndex();

function toVsCodeRange(range) {
    return new vscode.Range(
        new vscode.Position(range.start.line, range.start.character),
        new vscode.Position(range.end.line, range.end.character)
    );
}

function toVsCodeLocation(uri, range) {
    const targetUri = typeof uri === 'string' ? vscode.Uri.file(uri) : uri;
    return new vscode.Location(targetUri, toVsCodeRange(range));
}

function getSymbolKind(kind) {
    switch (kind) {
        case 'unit':
            return vscode.SymbolKind.Class;
        case 'function':
            return vscode.SymbolKind.Function;
        case 'argument':
            return vscode.SymbolKind.Property;
        case 'variable':
            return vscode.SymbolKind.Variable;
        case 'property':
            return vscode.SymbolKind.Field;
        case 'parameter':
            return vscode.SymbolKind.Variable;
        case 'module':
            return vscode.SymbolKind.Module;
        case 'directive':
            return vscode.SymbolKind.Method;
        default:
            return vscode.SymbolKind.Variable;
    }
}

/**
 * Scan workspace for all .xru files to build global index
 */
async function indexAllWorkspaceFiles() {
    try {
        const xruFiles = await vscode.workspace.findFiles('**/*.xru', '**/node_modules/**');
        for (const fileUri of xruFiles) {
            try {
                const doc = await vscode.workspace.openTextDocument(fileUri);
                workspaceIndex.indexDocument(fileUri.fsPath, doc.getText());
            } catch (err) {
                // Ignore unreadable files
            }
        }
    } catch (err) {
        // Workspace not open or search unavailable
    }
}

function activate(context) {
    // 1. Initial indexing
    indexAllWorkspaceFiles();

    // 2. Watch document lifecycle
    context.subscriptions.push(
        vscode.workspace.onDidOpenTextDocument(doc => {
            if (doc.languageId === 'xru' || doc.fileName.endsWith('.xru')) {
                workspaceIndex.indexDocument(doc.fileName, doc.getText());
            }
        })
    );

    context.subscriptions.push(
        vscode.workspace.onDidChangeTextDocument(event => {
            const doc = event.document;
            if (doc.languageId === 'xru' || doc.fileName.endsWith('.xru')) {
                workspaceIndex.indexDocument(doc.fileName, doc.getText());
            }
        })
    );

    context.subscriptions.push(
        vscode.workspace.onDidCloseTextDocument(doc => {
            // Keep in index for cross-file references unless deleted
        })
    );

    // 3. Definition Provider (Ctrl+Click / F12 "Go to Definition")
    const definitionProvider = vscode.languages.registerDefinitionProvider('xru', {
        provideDefinition(document, position, token) {
            const fsPath = document.fileName;
            // Always ensure the active document is up to date
            workspaceIndex.indexDocument(fsPath, document.getText());

            const def = workspaceIndex.resolveDefinition(fsPath, position.line, position.character);
            if (!def) {
                return null;
            }

            return toVsCodeLocation(def.uri, def.range);
        }
    });
    context.subscriptions.push(definitionProvider);

    // 4. Hover Provider
    const hoverProvider = vscode.languages.registerHoverProvider('xru', {
        provideHover(document, position, token) {
            const fsPath = document.fileName;
            workspaceIndex.indexDocument(fsPath, document.getText());

            const hover = workspaceIndex.resolveHover(fsPath, position.line, position.character);
            if (!hover) {
                return null;
            }

            const md = new vscode.MarkdownString(hover.contents);
            md.isTrusted = true;
            return new vscode.Hover(md);
        }
    });
    context.subscriptions.push(hoverProvider);

    // 5. Document Highlight Provider (Highlight all occurrences when cursor is on a variable)
    const highlightProvider = vscode.languages.registerDocumentHighlightProvider('xru', {
        provideDocumentHighlights(document, position, token) {
            const fsPath = document.fileName;
            const docAnalysis = workspaceIndex.indexDocument(fsPath, document.getText());

            const wordInfo = docAnalysis.getWordAt(position.line, position.character);
            if (!wordInfo) {
                return [];
            }

            const targetWord = wordInfo.word.toLowerCase();
            const highlights = [];

            for (const ref of docAnalysis.allReferences) {
                if (ref.name.toLowerCase() === targetWord) {
                    highlights.push(new vscode.DocumentHighlight(
                        toVsCodeRange(ref.range),
                        vscode.DocumentHighlightKind.Read
                    ));
                }
            }

            return highlights;
        }
    });
    context.subscriptions.push(highlightProvider);

    // 6. Reference Provider (Shift+F12 / "Find All References")
    const referenceProvider = vscode.languages.registerReferenceProvider('xru', {
        provideReferences(document, position, context, token) {
            const fsPath = document.fileName;
            workspaceIndex.indexDocument(fsPath, document.getText());

            const refs = workspaceIndex.findReferences(fsPath, position.line, position.character);
            return refs.map(r => toVsCodeLocation(r.uri, r.range));
        }
    });
    context.subscriptions.push(referenceProvider);

    // 7. Document Symbol Provider (Outline & Breadcrumbs)
    const symbolProvider = vscode.languages.registerDocumentSymbolProvider('xru', {
        provideDocumentSymbols(document, token) {
            const fsPath = document.fileName;
            const docAnalysis = workspaceIndex.indexDocument(fsPath, document.getText());

            const symbols = [];
            for (const s of docAnalysis.allSymbols) {
                const vsRange = toVsCodeRange(s.fullRange);
                const vsSelectionRange = toVsCodeRange(s.defRange);
                const docSymbol = new vscode.DocumentSymbol(
                    s.name,
                    s.detail,
                    getSymbolKind(s.kind),
                    vsRange,
                    vsSelectionRange
                );
                symbols.push(docSymbol);
            }

            return symbols;
        }
    });
    context.subscriptions.push(symbolProvider);
}

function deactivate() {
    workspaceIndex.documents.clear();
}

module.exports = {
    activate,
    deactivate
};
