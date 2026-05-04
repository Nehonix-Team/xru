import os
import re

directory = "xru/cmd/xru"

ast_types = [
    "RuleFile", "Rule", "RuleTypeBegin", "RuleTypeCreate", "RuleTypeSelect", "RuleTypeBreak", "RuleTypeLog",
    "RuleTypeIf", "RuleTypeElseIf", "RuleTypeElse", "RuleTypeInclude", "RuleTypeExec", "RuleTypeGlobal",
    "RuleTypeVar", "RuleTypeUse", "RuleTypeModule", "RuleTypeArg", "RuleTypeFor", "RuleTypeCall", "RuleTypeVarBlock",
    "PatchOp", "PatchMerge", "PatchSet", "PatchRM", "PatchPush", "PatchRPK", "PatchRPV", "PatchAppend", "PatchRegex",
    "Value", "Literal", "Object", "Array", "Action", "PatchAction", "InjectAction", "VarAction", "ModuleAction"
]

parser_funcs = ["Parse", "ParseFile", "ParseValue"]
utils_funcs = ["Interpolate", "InterpolateValue", "Dedent"]
inject_funcs = ["InjectCode"]
patcher_funcs = ["ApplyPatch"] # Wait, patcher functions will be in engine/patcher

for filename in os.listdir(directory):
    if filename.endswith(".go"):
        filepath = os.path.join(directory, filename)
        with open(filepath, 'r') as f:
            content = f.read()
        
        # update imports
        content = content.replace('"github.com/Nehonix-Team/xru/internal/engine"', 
"""	"github.com/Nehonix-Team/xru/internal/engine"
	"github.com/Nehonix-Team/xru/internal/engine/ast"
	"github.com/Nehonix-Team/xru/internal/engine/parser"
	"github.com/Nehonix-Team/xru/internal/engine/utils" """)

        # replace engine.Type with ast.Type
        for t in ast_types:
            content = content.replace(f"engine.{t}", f"ast.{t}")
        
        # replace engine.Func with parser.Func
        for f_name in parser_funcs:
            content = content.replace(f"engine.{f_name}", f"parser.{f_name}")
            
        # replace engine.Func with utils.Func
        for f_name in utils_funcs:
            content = content.replace(f"engine.{f_name}", f"utils.{f_name}")

        with open(filepath, 'w') as f:
            f.write(content)
