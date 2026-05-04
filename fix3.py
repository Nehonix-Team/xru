import os

directory = "/home/idevo/Documents/projects/XyPriss/tools/xru/cmd/xru"

def fix_file(filename, to_remove, to_add=None, replacements=None):
    filepath = os.path.join(directory, filename)
    with open(filepath, 'r') as f:
        content = f.read()

    lines = content.split('\n')
    new_lines = []
    for line in lines:
        skip = False
        for rem in to_remove:
            if rem in line:
                skip = True
                break
        if not skip:
            new_lines.append(line)
            
    content = '\n'.join(new_lines)
    
    if to_add:
        # insert after "github.com/Nehonix-Team/xru/internal/engine/ast" or similar
        content = content.replace('"github.com/Nehonix-Team/xru/internal/engine/ast"', '"github.com/Nehonix-Team/xru/internal/engine/ast"\n\t' + '\n\t'.join(to_add))
        
    if replacements:
        for old, new in replacements:
            content = content.replace(old, new)
            
    with open(filepath, 'w') as f:
        f.write(content)

fix_file("actions.go", [], 
    ['"github.com/Nehonix-Team/xru/internal/engine"', '"github.com/Nehonix-Team/xru/internal/engine/patcher"'],
    [('engine.ApplyPatch', 'patcher.ApplyPatch')])

fix_file("modules.go", ['"github.com/Nehonix-Team/xru/internal/engine/ast"', '"github.com/Nehonix-Team/xru/internal/engine/parser"'])
fix_file("scope.go", ['"github.com/Nehonix-Team/xru/internal/engine/parser"', '"github.com/Nehonix-Team/xru/internal/engine/util"'])
