# XRU Tutorial: Full Project Initialization (A-Z)

This tutorial demonstrates how to build a complete project initialization script using XRU. We will create a script that sets up a new web project with multiple environments, handles configuration patching, and performs code injection.

---

## 1. The Scenario
We want a script `init.xru` that:
1. Takes a **Project Name** as a positional argument.
2. Takes an **Environment** flag (`--env`).
3. Creates a standard directory structure.
4. Generates a `package.json` and a `config.ts`.
5. Patches a `README.md` if it exists.
6. Initializes a Git repository.

---

## 2. The Complete Script (`init.xru`)

```xru
// --------------------------------------------------------------------------
// 1. ARGUMENT PARSING & INITIALIZATION
// --------------------------------------------------------------------------
#ARG: 1 as PROJECT_NAME
#ARG: "--env" as ENV

#IF: {PROJECT_NAME} == ""
    U.LOG: "<red>[ERROR]</> Missing project name. Usage: xru init.xru <name> --env=<prod|dev>"
    U.EXIT: 1
#END

#IF: {ENV} == ""
    let ENV = "dev" // Default to development
#END

U.LOG: "<cyan>[INFO]</> Initializing project <white>{PROJECT_NAME}</> in <yellow>{ENV}</> mode..."

// --------------------------------------------------------------------------
// 2. WORKSPACE SETUP
// --------------------------------------------------------------------------
#IF: !exists({PROJECT_NAME})
    U.LOG: "Creating project directory..."
    FS.MKDIR: "{PROJECT_NAME}/src"
#END

#SELECT: "{PROJECT_NAME}" as ROOT

// --------------------------------------------------------------------------
// 3. FILE GENERATION (CREATE)
// --------------------------------------------------------------------------
#CREATE: "package.json"
{
  "name": "{PROJECT_NAME}",
  "version": "1.0.0",
  "scripts": {
    "start": "node dist/index.js"
  }
}
#END

#CREATE: "src/config.ts"
export const config = {
    env: "{ENV}",
    version: "1.0.0"
};

// --> {{extra_config}}
#END

// --------------------------------------------------------------------------
// 4. CONDITIONAL PATCHING
// --------------------------------------------------------------------------
#IF: {ENV} == "prod"
    #BEGIN: "package.json"
        SET version "1.0.0-release"
        MERGE scripts { "build": "tsc" }
    #END

    #BEGIN: "src/config.ts"
        @TSINJECT: extra_config
            export const analyticsId = "UA-XXXXXXXXX-1";
        @END
    #END
#ELSE:
    #BEGIN: "package.json"
        SET version "1.0.0-dev"
        PUSH keywords "draft"
    #END
#END

// --------------------------------------------------------------------------
// 5. GLOBAL OPERATIONS
// --------------------------------------------------------------------------
#IF: exists("README.md")
    #BEGIN: "README.md"
        ~~ "PROJECT_NAME_PLACEHOLDER" "{PROJECT_NAME}"
    #END
#ELSE:
    #CREATE: "README.md"
        # {PROJECT_NAME}
        Environment: {ENV}
    #END
#END

// --------------------------------------------------------------------------
// 6. EXTERNAL SYSTEM CALLS
// --------------------------------------------------------------------------
U.LOG: "Initializing Git repository..."
S.EXEC: "git init"

U.LOG: "<green>[SUCCESS]</> Project {PROJECT_NAME} is ready at <white>{ROOT}</>"
```

---

## 3. How to run this tutorial

### Development Mode
```bash
xru init.xru my-app --env=dev
```
- Creates `my-app` folder.
- Sets `env` to `"dev"` in `src/config.ts`.
- Adds `"1.0.0-dev"` version to `package.json`.

### Production Mode
```bash
xru init.xru my-app-prod --env=prod
```
- Creates `my-app-prod` folder.
- Injects `analyticsId` into `src/config.ts` via `@TSINJECT`.
- Adds `"build"` script to `package.json` via `MERGE`.

---

## 4. Syntax Summary (A-Z)

| Feature | Usage in Tutorial |
| :--- | :--- |
| **#ARG** | Captures `PROJECT_NAME` and `ENV`. |
| **#IF / #ELSE** | Handles default values and environment branching. |
| **let** | Defines the default `ENV`. |
| **FS.MKDIR** | Creates the project structure. |
| **#SELECT** | Scopes all following operations to the new project folder. |
| **#CREATE** | Generates the base `package.json` and `src/config.ts`. |
| **#BEGIN** | Patches files based on the selected environment. |
| **SET / MERGE** | Modifies JSON keys in `package.json`. |
| **@TSINJECT** | Performs precise code injection in `src/config.ts`. |
| **~~ (REGEX)** | Replaces placeholders in `README.md`. |
| **S.EXEC** | Runs `git init` in the project directory. |
| **U.LOG** | Provides colored feedback to the user. |
| **Interpolation** | Dynamically injects `{PROJECT_NAME}`, `{ENV}`, and `{ROOT}`. |

---

## 5. Advanced: Compact Orchestration
For complex architectures (like microservices), XRU allows merging data and templates into a single, high-density file using `#JSONVAR` and object iteration.

### The Problem: Multi-file redundancy
Usually, you would have a `.json` for data and multiple `.xru` for logic. For 5 servers, you might end up with many files.

### The Solution: Compact single-file approach
```xru
#USE: fs as FS
#SELECT: "./src/"

// 1. EMBEDDED DATA
#JSONVAR: SERVERS
{
  "auth": { "route": "/auth", "port": 8001 },
  "main": { "route": "/api",  "port": 8002, "extra": "cors: true" }
}
#END

// 2. DYNAMIC GENERATION
#FOR: S in {SERVERS}
    #CREATE: "servers/{S}.server.ts"
        @TSINJECT: ""
            export const {S}Server = {
                id: "{S}",
                port: {SERVERS.{S}.port},
                route: "{SERVERS.{S}.route}",
                {SERVERS.{S}.extra}
            };
        @END
    #END
#END
```

### Key Techniques used here:
- **`#JSONVAR`**: Embeds structured data directly in the script.
- **`#FOR: S in {SERVERS}`**: Iterates over object keys (`auth`, `main`).
- **Dot notation**: Accesses nested properties like `{SERVERS.{S}.port}`.
- **Automatic Induction**: Recursive interpolation allows using `{S}` to access `{SERVERS.{S}}`.
