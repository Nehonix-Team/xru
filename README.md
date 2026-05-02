# XRU: XyPriss Rule Unit

XRU is a Domain-Specific Language (DSL) designed for **Structured Text Transformation**. It operates as a **Structured Text Patcher (STP)**, preserving formatting, comments, and non-standard syntaxes while applying complex mutations.

---

## Quick Install

### Linux / macOS
```bash
curl -sL https://raw.githubusercontent.com/Nehonix-Team/xru/master/install.sh | bash
```

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/Nehonix-Team/xru/master/install.ps1 | iex
```

---

## Documentation

The documentation is modularized for better maintainability:

1.  **[Syntax Overview](./doc/syntax.md)**: General rules and log colorization.
2.  **[Directives](./doc/directives.md)**: Scoping (`#BEGIN`, `#SELECT`) and Utility (`#LOG`, `#EXEC`) directives.
3.  **[Actions](./doc/actions.md)**: Patching operations, symbols (`++`, `>>`), and code injections.
4.  **[Variables](./doc/variables.md)**: Scoping rules, declarations, and interpolation.
5.  **[CLI Usage](./doc/cli.md)**: Command line options and examples.

---

## Usage Example

Create a file named `patch.xru`:

```xru
let app = "MyProject"

#SELECT: ./src
#LOG: "<cyan>[INFO]</> Patching {app}..."

#BEGIN: config.json
  SET version "1.0.0"
  MERGE metadata {
    "author": "XyPriss"
  }
#END

#LOG: "<green>[SUCCESS]</> Done."
```

Apply it:
```bash
xru patch.xru .
```

---

## VS Code Support

For syntax highlighting, install the XRU extension located in `pkg/ext/vscode`.

---

## License

MIT © Nehonix-Team
