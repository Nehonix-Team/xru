import os
import re
import subprocess

def replace_in_file(filepath, replacements):
    with open(filepath, 'r') as f:
        content = f.read()
    
    for old, new in replacements:
        content = content.replace(old, new)
        
    with open(filepath, 'w') as f:
        f.write(content)

# Fix util package in internal/engine/util
replace_in_file("/home/idevo/Documents/projects/XyPriss/tools/xru/internal/engine/util/util.go", [
    ("package utils", "package util")
])

# Fix parser
replace_in_file("/home/idevo/Documents/projects/XyPriss/tools/xru/internal/engine/parser/parser.go", [
    ('"github.com/Nehonix-Team/xru/internal/engine/utils"', '"github.com/Nehonix-Team/xru/internal/engine/util"'),
    ('utils.Dedent', 'util.Dedent')
])

# Fix cmd/xru
directory = "/home/idevo/Documents/projects/XyPriss/tools/xru/cmd/xru"
for filename in os.listdir(directory):
    if filename.endswith(".go"):
        filepath = os.path.join(directory, filename)
        replace_in_file(filepath, [
            ('"github.com/Nehonix-Team/xru/internal/engine/utils"', '"github.com/Nehonix-Team/xru/internal/engine/util"'),
            ('utils.Interpolate', 'util.Interpolate'),
            ('utils.InterpolateValue', 'util.InterpolateValue'),
            ('utils.Dedent', 'util.Dedent'),
        ])
        
# Revert BinVersion to utils.BinVersion (Wait, we just replaced utils.Interpolate to util.Interpolate, so utils.BinVersion is untouched! Excellent)

# Clean up unused imports in cmd/xru using goimports if possible, or just build
# Actually we can just run `goimports -w .` in cmd/xru/
subprocess.run(["goimports", "-w", "."], cwd=directory)
