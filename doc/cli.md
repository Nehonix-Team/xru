# CLI Usage

The `xru` binary is a standalone tool for applying transformations.

## Usage
```bash
xru [options] <rule_file.xru> [target_directory]
```

- `<rule_file.xru>`: The file containing XRU instructions.
- `[target_directory]`: Optional. The root directory where rules are applied (defaults to `.`).

## Options
- `-v`, `--verbose`: Show detailed logs, including skipped blocks and path resolutions.
- `version`: Display version, OS, and Architecture info.
- `upgrade`: Automatically download and install the latest version for your platform.

## Examples
```bash
# Run rules in the current directory
xru update.xru

# Run rules on a specific project folder
xru patch.xru ./my-project

# Passing arguments to a script
xru script.xru --mode=prod --debug

# Passing both target and arguments
xru script.xru ./target-dir --mode=prod
```

### Argument Detection Logic
XRU uses a smart detection for the `[target_directory]`:
- If the argument after the rule file starts with `-` (a flag), XRU assumes the target is `.` and treats that argument as a script argument.
- Otherwise, it treats the first argument as the target and subsequent arguments as script arguments.

#### Script Implementation of Arguments
```xru
// script.xru
#ARG: "--mode" as MODE
#ARG: 1 as TARGET_FILE

#IF: {MODE} == "prod"
    U.LOG: "Processing {TARGET_FILE} in PRODUCTION mode"
#END
```

