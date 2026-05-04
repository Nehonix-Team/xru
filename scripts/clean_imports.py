import os

directory = "xru/cmd/xru"

for filename in os.listdir(directory):
    if filename.endswith(".go"):
        filepath = os.path.join(directory, filename)
        with open(filepath, 'r') as f:
            lines = f.readlines()
            
        new_lines = []
        for line in lines:
            if '"github.com/Nehonix-Team/xru/internal/engine"' in line: continue
            
            if filename == "actions.go" and '"github.com/Nehonix-Team/xru/internal/engine/parser"' in line: continue
            if filename == "build.go":
                if '"github.com/Nehonix-Team/xru/internal/engine/ast"' in line or '"github.com/Nehonix-Team/xru/internal/engine/util"' in line:
                    continue
            if filename == "condition.go":
                if '"github.com/Nehonix-Team/xru/internal/engine/ast"' in line or '"github.com/Nehonix-Team/xru/internal/engine/parser"' in line:
                    continue
            if filename == "main.go":
                if '"github.com/Nehonix-Team/xru/internal/engine/ast"' in line or '"github.com/Nehonix-Team/xru/internal/engine/util"' in line:
                    continue
            
            new_lines.append(line)
            
        with open(filepath, 'w') as f:
            f.writelines(new_lines)

# Also fix the utils redeclaration issue in cmd/xru/main.go
with open(os.path.join(directory, "main.go"), 'r') as f:
    content = f.read()

# BinVersion is imported from github.com/Nehonix-Team/xru/internal/utils.
# If "github.com/Nehonix-Team/xru/internal/utils" isn't imported, add it.
if '"github.com/Nehonix-Team/xru/internal/utils"' not in content:
    content = content.replace('"github.com/Nehonix-Team/xru/internal/engine/parser"', '"github.com/Nehonix-Team/xru/internal/engine/parser"\n\t"github.com/Nehonix-Team/xru/internal/utils"')
    
with open(os.path.join(directory, "main.go"), 'w') as f:
    f.write(content)
