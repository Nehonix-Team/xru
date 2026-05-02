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

# Verbose execution for debugging
xru -v setup.xru
```
