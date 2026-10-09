/**
 * XRU 2.0 Semantic Analyzer & AST Indexer for VS Code
 * Provides Go-to-Definition, Hover, Document Highlights, and References
 */

class Position {
    constructor(line, character) {
        this.line = line;
        this.character = character;
    }
}

class Range {
    constructor(startLine, startCol, endLine, endCol) {
        this.start = new Position(startLine, startCol);
        this.end = new Position(endLine, endCol);
    }
}

class SymbolDef {
    constructor(name, kind, type, detail, doc, defRange, fullRange, uri = null) {
        this.name = name;
        this.kind = kind; // 'variable', 'argument', 'function', 'unit', 'parameter', 'module', 'property', 'directive'
        this.type = type || 'any';
        this.detail = detail || `${kind} ${name}`;
        this.doc = doc || '';
        this.defRange = defRange; // Range of identifier
        this.fullRange = fullRange || defRange;
        this.uri = uri;
    }
}

class Scope {
    constructor(parent = null, range = null, name = 'block') {
        this.parent = parent;
        this.children = [];
        this.range = range; // Range covering this scope block
        this.name = name;
        this.symbols = new Map(); // lowercase name -> SymbolDef
        if (parent) {
            parent.children.push(this);
        }
    }

    addSymbol(sym) {
        this.symbols.set(sym.name.toLowerCase(), sym);
    }

    getSymbol(name) {
        const key = name.toLowerCase();
        if (this.symbols.has(key)) {
            return this.symbols.get(key);
        }
        if (this.parent) {
            return this.parent.getSymbol(name);
        }
        return null;
    }

    findScopeAt(line, character) {
        for (const child of this.children) {
            if (child.range && isPositionInRange(line, character, child.range)) {
                return child.findScopeAt(line, character);
            }
        }
        return this;
    }
}

function isPositionInRange(line, character, range) {
    if (line < range.start.line || line > range.end.line) return false;
    if (line === range.start.line && character < range.start.character) return false;
    if (line === range.end.line && character > range.end.character) return false;
    return true;
}

// Built-in standard library knowledge
const BUILTINS = {
    units: {
        'Utils': {
            doc: 'Standard logging and terminal formatting utility unit.',
            methods: {
                'log': {
                    detail: 'Utils.log(message: any) -> void',
                    doc: 'Prints a formatted message to the terminal with ANSI colors support (e.g. `<green>...</>`, `<red>...</>`, `<cyan>...</>`).'
                },
                'getRandomBytes': {
                    detail: 'Utils.getRandomBytes(length: number = 32) -> string',
                    doc: 'Generates a cryptographically secure random hexadecimal string.'
                }
            }
        },
        'Net': {
            doc: 'HTTP networking client and remote package download engine.',
            methods: {
                'fetch': {
                    detail: 'Net.fetch(url: string) -> FetchContext',
                    doc: 'Downloads an asset or release from a remote URL with real-time streaming progress, speed computation, and optional auto-extraction.'
                }
            }
        },
        'Crypto': {
            doc: 'Cryptographic primitives unit for hashing, UUID, and secure randomness.',
            methods: {
                'getRandomBytes': {
                    detail: 'Crypto.getRandomBytes(length: number = 32) -> string',
                    doc: 'Generates cryptographically secure random bytes formatted as hex.'
                },
                'randomHex': {
                    detail: 'Crypto.randomHex(length: number = 32) -> string',
                    doc: 'Generates random hexadecimal characters.'
                },
                'sha256': {
                    detail: 'Crypto.sha256(data: string) -> string',
                    doc: 'Computes the SHA-256 hash of a string.'
                },
                'uuid': {
                    detail: 'Crypto.uuid() -> string',
                    doc: 'Generates a random UUID v4 identifier.'
                }
            }
        },
        'Fs': {
            doc: 'File system interaction unit.',
            methods: {
                'exists': {
                    detail: 'Fs.exists(path: string) -> bool',
                    doc: 'Checks whether the specified file or directory path exists.'
                },
                'remove': {
                    detail: 'Fs.remove(path: string) -> bool',
                    doc: 'Deletes a file or directory recursively.'
                },
                'move': {
                    detail: 'Fs.move(source: string, destination: string) -> bool',
                    doc: 'Renames or moves a file to destination.'
                },
                'read': {
                    detail: 'Fs.read(path: string) -> string',
                    doc: 'Reads the entire contents of a file as a string.'
                },
                'write': {
                    detail: 'Fs.write(path: string, content: string) -> bool',
                    doc: 'Writes text contents to a file, creating parent directories if needed.'
                }
            }
        },
        'Local': {
            doc: 'Local module sandboxing unit used inside subdescriptors (.rules.xru).',
            methods: {
                'Keep': {
                    detail: 'Local.Keep(files: list[string]) -> void',
                    doc: 'Keeps only the specified files in current directory and drops all others.'
                },
                'Drop': {
                    detail: 'Local.Drop(patterns: list[string]) -> void',
                    doc: 'Drops files matching the specified patterns in current directory.'
                }
            }
        }
    },
    fileMethods: {
        'replace': {
            detail: 'replace(target: string, replacement: string)',
            doc: 'Replaces all occurrences of `target` with `replacement` in the file.'
        },
        'replaceLine': {
            detail: 'replaceLine(pattern: string, replacement: string)',
            doc: 'Replaces any line containing `pattern` with `replacement`.'
        },
        'remove': {
            detail: 'remove(target: string)',
            doc: 'Deletes all occurrences of `target` string in the file.'
        },
        'removeLines': {
            detail: 'removeLines(pattern: string)',
            doc: 'Deletes all lines containing the `pattern`.'
        },
        'removeBlock': {
            detail: 'removeBlock(startPattern: string, endPattern: string)',
            doc: 'Deletes the entire block of code starting at `startPattern` through `endPattern`.'
        },
        'removeUntil': {
            detail: 'removeUntil(startPattern: string, untilPattern: string)',
            doc: 'Deletes lines from `startPattern` up until `untilPattern`.'
        },
        'commentOut': {
            detail: 'commentOut(pattern: string)',
            doc: 'Prepends lines containing `pattern` with `//` comment markers.'
        },
        'append': {
            detail: 'append(content: string)',
            doc: 'Appends content at the end of the file.'
        },
        'prepend': {
            detail: 'prepend(content: string)',
            doc: 'Prepends content at the top of the file.'
        },
        'jsonSet': {
            detail: 'jsonSet(keyPath: string, value: any)',
            doc: 'Modifies or inserts a JSON property by dot-separated path (e.g. `scripts.build`).'
        },
        'jsonRemove': {
            detail: 'jsonRemove(key: string)',
            doc: 'Removes a key from a JSON file.'
        }
    },
    globalFunctions: {
        'Drop': {
            detail: 'Drop(patterns: list[string])',
            doc: 'Prunes / drops files in the current directory matching specified patterns.'
        },
        'drop': {
            detail: 'drop(patterns: list[string])',
            doc: 'Prunes / drops files in the current directory matching specified patterns.'
        }
    }
};

class DocumentAnalysis {
    constructor(text, uri = null) {
        this.text = text;
        this.uri = uri;
        this.lines = text.split(/\r?\n/);
        this.rootScope = new Scope(null, new Range(0, 0, this.lines.length, 0), 'global');
        this.allSymbols = [];
        this.allReferences = []; // { name, range, isDef }
        this.parse();
    }

    parse() {
        const lines = this.lines;
        let currentScope = this.rootScope;
        const scopeStack = [currentScope];

        // Track braces and line blocks
        for (let lineIdx = 0; lineIdx < lines.length; lineIdx++) {
            const rawLine = lines[lineIdx];
            const trimmed = rawLine.trim();

            // Skip pure comment lines
            if (trimmed.startsWith('//')) {
                continue;
            }

            // 1. Module import: use <module> as <alias> / use <module>
            const useMatch = rawLine.match(/^(\s*)use\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s+as\s+([a-zA-Z_][a-zA-Z0-9_]*))?/);
            if (useMatch) {
                const indent = useMatch[1].length;
                const modName = useMatch[2];
                const alias = useMatch[3] || modName;
                const nameCol = rawLine.indexOf(alias, indent);
                const defRange = new Range(lineIdx, nameCol, lineIdx, nameCol + alias.length);
                const fullRange = new Range(lineIdx, indent, lineIdx, rawLine.length);
                const sym = new SymbolDef(
                    alias,
                    'module',
                    'module',
                    `use ${modName}${useMatch[3] ? ' as ' + alias : ''}`,
                    `Imported module '${modName}' aliased as '${alias}'.`,
                    defRange,
                    fullRange,
                    this.uri
                );
                currentScope.addSymbol(sym);
                this.allSymbols.push(sym);
            }

            // 2. Arguments: arg <name>: <type> = <default> / arg <name>: <type> or <default> / arg <name> = <default>
            const argMatch = rawLine.match(/^(\s*)arg\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s*:\s*([a-zA-Z0-9_\[\], ]+?))?(?:\s*(?:=|or)\s*(.*))?$/);
            if (argMatch) {
                const indent = argMatch[1].length;
                const name = argMatch[2];
                const type = argMatch[3] || 'any';
                const defaultVal = argMatch[4] ? ` = ${argMatch[4].trim()}` : '';
                const nameCol = rawLine.indexOf(name, indent);
                const defRange = new Range(lineIdx, nameCol, lineIdx, nameCol + name.length);
                const fullRange = new Range(lineIdx, indent, lineIdx, rawLine.length);
                const sym = new SymbolDef(
                    name,
                    'argument',
                    type,
                    `arg ${name}: ${type}${defaultVal}`,
                    `CLI argument passed via \`--${name}=<value>\` (default: ${argMatch[4] ? argMatch[4].trim() : 'none'}).`,
                    defRange,
                    fullRange,
                    this.uri
                );
                currentScope.addSymbol(sym);
                this.allSymbols.push(sym);
            }

            // 3. Variable declaration: let/const <name>: <type> = <value>
            const varMatch = rawLine.match(/^(\s*)(let|const)\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s*:\s*([a-zA-Z0-9_\[\], ]+?))?(?:\s*=\s*(.*))?$/);
            if (varMatch) {
                const indent = varMatch[1].length;
                const keyword = varMatch[2];
                const name = varMatch[3];
                const type = varMatch[4] || 'inferred';
                const initVal = varMatch[5] ? ` = ${varMatch[5].trim()}` : '';
                const nameCol = rawLine.indexOf(name, indent);
                const defRange = new Range(lineIdx, nameCol, lineIdx, nameCol + name.length);
                const fullRange = new Range(lineIdx, indent, lineIdx, rawLine.length);
                const sym = new SymbolDef(
                    name,
                    'variable',
                    type,
                    `${keyword} ${name}${varMatch[4] ? ': ' + type : ''}${initVal}`,
                    `${keyword === 'const' ? 'Constant' : 'Variable'} declaration.`,
                    defRange,
                    fullRange,
                    this.uri
                );
                currentScope.addSymbol(sym);
                this.allSymbols.push(sym);
            }

            // 4. Property declaration inside units: declare <name>: <type> = <value>
            const declMatch = rawLine.match(/^(\s*)declare\s+([a-zA-Z_][a-zA-Z0-9_]*)(?:\s*:\s*([a-zA-Z0-9_\[\], ]+?))?(?:\s*=\s*(.*))?$/);
            if (declMatch) {
                const indent = declMatch[1].length;
                const name = declMatch[2];
                const type = declMatch[3] || 'any';
                const nameCol = rawLine.indexOf(name, indent);
                const defRange = new Range(lineIdx, nameCol, lineIdx, nameCol + name.length);
                const fullRange = new Range(lineIdx, indent, lineIdx, rawLine.length);
                const sym = new SymbolDef(
                    name,
                    'property',
                    type,
                    `declare ${name}: ${type}`,
                    `Internal unit property accessible via \`&${name}\`.`,
                    defRange,
                    fullRange,
                    this.uri
                );
                currentScope.addSymbol(sym);
                this.allSymbols.push(sym);
            }

            // 5. Unit declaration: unit <name> {
            const unitMatch = rawLine.match(/^(\s*)unit\s+([a-zA-Z_][a-zA-Z0-9_]*)/);
            if (unitMatch) {
                const indent = unitMatch[1].length;
                const name = unitMatch[2];
                const nameCol = rawLine.indexOf(name, indent);
                const defRange = new Range(lineIdx, nameCol, lineIdx, nameCol + name.length);
                const sym = new SymbolDef(
                    name,
                    'unit',
                    'unit',
                    `unit ${name}`,
                    `Reusable custom unit. Instantiate via \`@${name}()\`.`,
                    defRange,
                    defRange,
                    this.uri
                );
                this.rootScope.addSymbol(sym);
                this.allSymbols.push(sym);
            }

            // 6. Function: func <name>(params) -> returnType / global func <name>(...)
            const funcMatch = rawLine.match(/^(\s*)(?:(global)\s+)?func\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\((.*?)\)(?:\s*->\s*([a-zA-Z0-9_\[\], ]+))?/);
            if (funcMatch) {
                const indent = funcMatch[1].length;
                const isGlobal = !!funcMatch[2];
                const name = funcMatch[3];
                const rawParams = funcMatch[4];
                const returnType = funcMatch[5] || 'void';
                const nameCol = rawLine.indexOf(name, indent);
                const defRange = new Range(lineIdx, nameCol, lineIdx, nameCol + name.length);
                const detail = `${isGlobal ? 'global ' : ''}func ${name}(${rawParams}) -> ${returnType}`;
                const doc = isGlobal 
                    ? `Global function. Can be invoked cross-module via \`::${name}(...)\` or \`global::${name}(...)\`.`
                    : `Local function.`;

                const sym = new SymbolDef(
                    name,
                    'function',
                    returnType,
                    detail,
                    doc,
                    defRange,
                    defRange,
                    this.uri
                );
                (isGlobal ? this.rootScope : currentScope).addSymbol(sym);
                this.allSymbols.push(sym);
            }

            // Check for Lambda parameters: (param1, param2) =>
            const lambdaMatch = rawLine.match(/(?:\(([^)]+)\)|([a-zA-Z_][a-zA-Z0-9_]*))\s*=>/);
            let pendingLambdaParams = [];
            if (lambdaMatch) {
                const paramStr = lambdaMatch[1] || lambdaMatch[2];
                if (paramStr) {
                    const params = paramStr.split(',').map(s => s.trim()).filter(Boolean);
                    for (const p of params) {
                        const pName = p.split(':')[0].trim();
                        if (pName) {
                            const pCol = rawLine.indexOf(pName);
                            const pRange = new Range(lineIdx, pCol, lineIdx, pCol + pName.length);
                            pendingLambdaParams.push(new SymbolDef(
                                pName,
                                'parameter',
                                'any',
                                `(parameter) ${pName}: any`,
                                `Callback lambda parameter.`,
                                pRange,
                                pRange,
                                this.uri
                            ));
                        }
                    }
                }
            }

            // Function parameters: if line defines func ..., extract params
            if (funcMatch && funcMatch[4]) {
                const params = funcMatch[4].split(',').map(s => s.trim()).filter(Boolean);
                for (const p of params) {
                    const parts = p.split(':');
                    const pName = parts[0].trim();
                    const pType = parts[1] ? parts[1].trim() : 'any';
                    const pCol = rawLine.indexOf(pName);
                    const pRange = new Range(lineIdx, pCol, lineIdx, pCol + pName.length);
                    pendingLambdaParams.push(new SymbolDef(
                        pName,
                        'parameter',
                        pType,
                        `(parameter) ${pName}: ${pType}`,
                        `Function parameter.`,
                        pRange,
                        pRange,
                        this.uri
                    ));
                }
            }

            // Scope handling by brace tracking (ignoring braces inside quotes or comments)
            let inString = null;
            let hasBraceInLine = false;

            for (let charIdx = 0; charIdx < rawLine.length; charIdx++) {
                const ch = rawLine[charIdx];
                if (!inString) {
                    if (ch === '/' && rawLine[charIdx + 1] === '/') {
                        break; // rest of line is comment
                    }
                    if (ch === '"' || ch === "'" || ch === '`') {
                        inString = ch;
                        continue;
                    }
                    if (ch === '{') {
                        hasBraceInLine = true;
                        const newScope = new Scope(currentScope, new Range(lineIdx, charIdx, lines.length, 0), 'block');
                        currentScope = newScope;
                        scopeStack.push(currentScope);

                        // If we have pending lambda/func params, register into this new scope
                        if (pendingLambdaParams.length > 0) {
                            for (const lp of pendingLambdaParams) {
                                currentScope.addSymbol(lp);
                                this.allSymbols.push(lp);
                            }
                            pendingLambdaParams = [];
                        }
                    } else if (ch === '}') {
                        if (scopeStack.length > 1) {
                            currentScope.range.end = new Position(lineIdx, charIdx);
                            scopeStack.pop();
                            currentScope = scopeStack[scopeStack.length - 1];
                        }
                    }
                } else {
                    if (ch === '\\') {
                        charIdx++; // skip escape
                        continue;
                    }
                    if (ch === inString) {
                        inString = null;
                    }
                }
            }

            // If lambda was inline without braces (e.g. `(p, s) => Utils.log(...)`),
            // create an inline scope covering the rest of this line
            if (pendingLambdaParams.length > 0) {
                if (!hasBraceInLine) {
                    const arrowIdx = rawLine.indexOf('=>');
                    const inlineScope = new Scope(currentScope, new Range(lineIdx, arrowIdx, lineIdx, rawLine.length), 'lambda');
                    for (const lp of pendingLambdaParams) {
                        inlineScope.addSymbol(lp);
                        this.allSymbols.push(lp);
                    }
                } else {
                    for (const lp of pendingLambdaParams) {
                        currentScope.addSymbol(lp);
                        this.allSymbols.push(lp);
                    }
                }
                pendingLambdaParams = [];
            }

            // Scan all identifiers and string interpolations in this line for references
            this.scanLineReferences(rawLine, lineIdx);
        }
    }

    scanLineReferences(line, lineIdx) {
        // 1. Identifiers
        const identRegex = /\b([a-zA-Z_][a-zA-Z0-9_]*)\b/g;
        let match;
        while ((match = identRegex.exec(line)) !== null) {
            const word = match[1];
            const startCol = match.index;
            const endCol = startCol + word.length;
            this.allReferences.push({
                name: word,
                range: new Range(lineIdx, startCol, lineIdx, endCol)
            });
        }

        // 2. String interpolations: "{varName}" or "{&propName}"
        const interpRegex = /\{&?([a-zA-Z_][a-zA-Z0-9_]*)\}/g;
        while ((match = interpRegex.exec(line)) !== null) {
            const word = match[1];
            // index of the inner word
            const innerStart = line.indexOf(word, match.index);
            const innerEnd = innerStart + word.length;
            this.allReferences.push({
                name: word,
                range: new Range(lineIdx, innerStart, lineIdx, innerEnd),
                isInterpolation: true
            });
        }
    }

    getWordAt(line, character) {
        if (line < 0 || line >= this.lines.length) return null;
        const lineText = this.lines[line];

        // Check if inside `{word}` interpolation
        const interpMatch = lineText.match(/\{&?([a-zA-Z_][a-zA-Z0-9_]*)\}/g);
        if (interpMatch) {
            for (const m of interpMatch) {
                const idx = lineText.indexOf(m);
                const word = m.replace(/^\{&?/, '').replace(/\}$/, '');
                const wordStart = lineText.indexOf(word, idx);
                const wordEnd = wordStart + word.length;
                if (character >= wordStart && character <= wordEnd) {
                    return {
                        word: word,
                        range: new Range(line, wordStart, line, wordEnd)
                    };
                }
            }
        }

        // Check if prefixed by `::` or `global::`
        const globalCallRegex = /(?:global::|::)([a-zA-Z_][a-zA-Z0-9_]*)/g;
        let gMatch;
        while ((gMatch = globalCallRegex.exec(lineText)) !== null) {
            const fnName = gMatch[1];
            const start = gMatch.index + (gMatch[0].startsWith('global::') ? 8 : 2);
            const end = start + fnName.length;
            if (character >= start && character <= end) {
                return {
                    word: fnName,
                    range: new Range(line, start, line, end),
                    isGlobalCall: true
                };
            }
        }

        // Standard identifier word detection
        let start = character;
        while (start > 0 && /[a-zA-Z0-9_]/.test(lineText[start - 1])) {
            start--;
        }
        let end = character;
        while (end < lineText.length && /[a-zA-Z0-9_]/.test(lineText[end])) {
            end++;
        }

        if (start === end) return null;
        const word = lineText.substring(start, end);
        return {
            word: word,
            range: new Range(line, start, line, end)
        };
    }

    resolveDefinition(line, character) {
        const wordInfo = this.getWordAt(line, character);
        if (!wordInfo) return null;
        const word = wordInfo.word;

        // 1. Check local scope upwards
        const targetScope = this.rootScope.findScopeAt(line, character);
        const localSym = targetScope.getSymbol(word);
        if (localSym) {
            return {
                symbol: localSym,
                range: localSym.defRange,
                uri: this.uri
            };
        }

        // 2. Check root scope
        const rootSym = this.rootScope.getSymbol(word);
        if (rootSym) {
            return {
                symbol: rootSym,
                range: rootSym.defRange,
                uri: this.uri
            };
        }

        return null;
    }

    resolveHover(line, character) {
        const wordInfo = this.getWordAt(line, character);
        if (!wordInfo) return null;
        const word = wordInfo.word;

        // Check user defined symbol
        const def = this.resolveDefinition(line, character);
        if (def && def.symbol) {
            return {
                contents: [
                    `\`\`\`xru\n${def.symbol.detail}\n\`\`\``,
                    def.symbol.doc
                ].filter(Boolean).join('\n\n')
            };
        }

        // Check Builtin Units
        if (BUILTINS.units[word]) {
            const u = BUILTINS.units[word];
            return {
                contents: [
                    `\`\`\`xru\nunit ${word}\n\`\`\``,
                    u.doc
                ].join('\n\n')
            };
        }

        // Check member call e.g. `Utils.log` or `Crypto.getRandomBytes`
        const lineText = this.lines[line];
        const memberMatch = lineText.match(new RegExp(`([a-zA-Z_][a-zA-Z0-9_]*)\\.${word}\\b`));
        if (memberMatch) {
            const unitName = memberMatch[1];
            if (BUILTINS.units[unitName] && BUILTINS.units[unitName].methods[word]) {
                const m = BUILTINS.units[unitName].methods[word];
                return {
                    contents: [
                        `\`\`\`xru\n${m.detail}\n\`\`\``,
                        m.doc
                    ].join('\n\n')
                };
            }
        }

        // Check File Surgical methods e.g. `replace`, `replaceLine`, etc.
        if (BUILTINS.fileMethods[word]) {
            const fm = BUILTINS.fileMethods[word];
            return {
                contents: [
                    `\`\`\`xru\n${fm.detail}\n\`\`\``,
                    fm.doc
                ].join('\n\n')
            };
        }

        // Check global functions like `Drop`
        if (BUILTINS.globalFunctions[word]) {
            const gf = BUILTINS.globalFunctions[word];
            return {
                contents: [
                    `\`\`\`xru\n${gf.detail}\n\`\`\``,
                    gf.doc
                ].join('\n\n')
            };
        }

        return null;
    }

    findReferences(line, character) {
        const wordInfo = this.getWordAt(line, character);
        if (!wordInfo) return [];
        const word = wordInfo.word.toLowerCase();

        const matches = [];
        for (const ref of this.allReferences) {
            if (ref.name.toLowerCase() === word) {
                matches.push({
                    range: ref.range,
                    uri: this.uri
                });
            }
        }
        return matches;
    }

    validate() {
        const diagnostics = [];
        const KNOWN_MODULES = new Set(['utils', 'fs', 'net', 'crypto', 'local', 'sys', 'path', 'cmd']);
        const NET_PROPERTIES = new Set(['method', 'timeout', 'extractto', 'dest']);
        const NET_EVENTS = new Set(['progress', 'complete']);

        let insideWithNet = false;
        let withBraceDepth = 0;

        for (let lineIdx = 0; lineIdx < this.lines.length; lineIdx++) {
            const rawLine = this.lines[lineIdx];
            const trimmed = rawLine.trim();

            if (trimmed.startsWith('//')) continue;

            // 1. Module import check
            const useMatch = rawLine.match(/^\s*use\s+([a-zA-Z_][a-zA-Z0-9_]*)/);
            if (useMatch) {
                const modName = useMatch[1];
                if (!KNOWN_MODULES.has(modName.toLowerCase())) {
                    const col = rawLine.indexOf(modName);
                    diagnostics.push({
                        range: new Range(lineIdx, col, lineIdx, col + modName.length),
                        message: `Module '${modName}' does not exist`,
                        severity: 'error'
                    });
                }
            }

            // 2. Track with Net.fetch(...)
            if (/^\s*with\s+Net\.fetch\b/i.test(rawLine)) {
                insideWithNet = true;
            }

            if (insideWithNet) {
                for (let c = 0; c < rawLine.length; c++) {
                    const ch = rawLine[c];
                    if (ch === '{') withBraceDepth++;
                    else if (ch === '}') {
                        withBraceDepth--;
                        if (withBraceDepth <= 0) {
                            insideWithNet = false;
                        }
                    }
                }

                // Check directives inside with Net
                const setMatch = rawLine.match(/^\s*set\s+([a-zA-Z_][a-zA-Z0-9_]*)/);
                if (setMatch) {
                    const propName = setMatch[1];
                    if (!NET_PROPERTIES.has(propName.toLowerCase())) {
                        const col = rawLine.indexOf(propName);
                        diagnostics.push({
                            range: new Range(lineIdx, col, lineIdx, col + propName.length),
                            message: `Property '${propName}' does not exist on Net.fetch (supported: method, timeout, extractTo, dest)`,
                            severity: 'error'
                        });
                    }
                }

                const onMatch = rawLine.match(/^\s*on\s+([a-zA-Z_][a-zA-Z0-9_]*)/);
                if (onMatch) {
                    const eventName = onMatch[1];
                    if (!NET_EVENTS.has(eventName.toLowerCase())) {
                        const col = rawLine.indexOf(eventName);
                        diagnostics.push({
                            range: new Range(lineIdx, col, lineIdx, col + eventName.length),
                            message: `Event '${eventName}' does not exist on Net.fetch (supported: progress, complete)`,
                            severity: 'error'
                        });
                    }
                }
            }
        }

        return diagnostics;
    }
}

class WorkspaceIndex {
    constructor() {
        this.documents = new Map(); // uri -> DocumentAnalysis
    }

    indexDocument(uri, text) {
        const doc = new DocumentAnalysis(text, uri);
        this.documents.set(uri, doc);
        return doc;
    }

    removeDocument(uri) {
        this.documents.delete(uri);
    }

    getDocument(uri) {
        return this.documents.get(uri);
    }

    resolveDefinition(uri, line, character) {
        const doc = this.documents.get(uri);
        if (!doc) return null;

        // 1. Try local scope first
        const localDef = doc.resolveDefinition(line, character);
        if (localDef) {
            return localDef;
        }

        const wordInfo = doc.getWordAt(line, character);
        if (!wordInfo) return null;
        const word = wordInfo.word;

        // 2. Cross-file search for global functions or root declarations
        for (const [docUri, otherDoc] of this.documents.entries()) {
            if (docUri === uri) continue;
            const sym = otherDoc.rootScope.getSymbol(word);
            if (sym) {
                return {
                    symbol: sym,
                    range: sym.defRange,
                    uri: docUri
                };
            }
        }

        return null;
    }

    resolveHover(uri, line, character) {
        const doc = this.documents.get(uri);
        if (!doc) return null;

        // Try local hover first
        const localHover = doc.resolveHover(line, character);
        if (localHover) return localHover;

        // Check if it's a cross-file global symbol
        const def = this.resolveDefinition(uri, line, character);
        if (def && def.symbol) {
            return {
                contents: [
                    `\`\`\`xru\n${def.symbol.detail}\n\`\`\``,
                    def.symbol.doc,
                    `*(defined in ${def.uri ? def.uri.split('/').pop() : 'external file'})*`
                ].filter(Boolean).join('\n\n')
            };
        }

        return null;
    }

    findReferences(uri, line, character) {
        const doc = this.documents.get(uri);
        if (!doc) return [];

        const wordInfo = doc.getWordAt(line, character);
        if (!wordInfo) return [];
        const word = wordInfo.word.toLowerCase();

        const allRefs = [];
        // Search across all indexed documents
        for (const [docUri, d] of this.documents.entries()) {
            for (const ref of d.allReferences) {
                if (ref.name.toLowerCase() === word) {
                    allRefs.push({
                        range: ref.range,
                        uri: docUri
                    });
                }
            }
        }
        return allRefs;
    }
}

module.exports = {
    Position,
    Range,
    SymbolDef,
    Scope,
    DocumentAnalysis,
    WorkspaceIndex,
    BUILTINS
};
