"""Real TSX/Svelte/Jest coverage acceptance; invoked by the Orrery plan."""
import json,os,subprocess,sys,tempfile
from pathlib import Path
cli=Path(sys.argv[1]).resolve();tools=Path(sys.argv[2]).resolve();out=Path(sys.argv[3]).resolve();out.mkdir(parents=True,exist_ok=True)
workspace=Path(__file__).resolve().parents[2]
def command(args):
 p=subprocess.run([str(a) for a in args],text=True,capture_output=True)
 assert p.returncode==0,(args,p.stdout[-2000:],p.stderr[-6000:])
 return p.stdout
command([cli,'engine','install','ts-js','--from',workspace/'scryer-ts-js-engine/build/install/scryer-ts-js-engine','--json'])
for case in ['react','svelte','jest']:
 with tempfile.TemporaryDirectory(prefix='scryer-'+case+'-') as folder:
  root=Path(folder).resolve()
  def write(name,text):
   p=root/name;p.parent.mkdir(parents=True,exist_ok=True);p.write_text(text)
  def git(*args):return command(['git','-C',root,*args]).strip()
  (root/'node_modules').symlink_to(tools/'node_modules',target_is_directory=True)
  write('.gitignore','node_modules\n.svelte-kit\ncoverage\n.vite\n')
  pkg={'name':'analysis-'+case,'private':True,'type':'module','scripts':{'test':'vitest run'},'devDependencies':{'vitest':'5.0.5','@vitest/coverage-v8':'5.0.5','typescript':'6.0.3'}}
  compiler={'target':'ES2022','module':'ESNext','moduleResolution':'Bundler','skipLibCheck':True,'allowJs':True,'checkJs':False,'jsx':'react-jsx','allowImportingTsExtensions':True,'noEmit':True}
  if case=='react':
   pkg['dependencies']={'react':'19.3.1'}
   source='export function value(n: number): number { if(n>0) return 1; return 0; }\nexport function caller() { return value(1); }\nexport function uncovered() { return 1; }\nexport function Panel() { return <section>{caller()}</section>; }\n'
   path='src/Panel.tsx';test='import {test,expect} from "vitest";\nimport {Panel} from "../src/Panel";\ntest("panel",()=>expect(Panel().type).toBe("section"));\n'
  elif case=='svelte':
   pkg['dependencies']={'svelte':'5.57.3'};pkg['devDependencies']['@sveltejs/vite-plugin-svelte']='7.3.1'
   source='<script module lang="ts">\nexport function value(n: number): number { if(n>0) return 1; return 0; }\nexport function caller() { return value(1); }\nexport function uncovered() { return 1; }\n</script>\n<p>fixture</p>\n';path='src/Counter.svelte'
   test='import {test,expect} from "vitest";\nimport {caller} from "../src/Counter.svelte";\ntest("counter",()=>expect(caller()).toBeGreaterThan(0));\n'
   write('src/env.d.ts','declare module "*.svelte" { export function caller(): number; const component: any; export default component; }\n')
   write('vitest.config.mjs','import {defineConfig} from "vitest/config";\nimport {svelte} from "@sveltejs/vite-plugin-svelte";\nexport default defineConfig({plugins:[svelte()]});\n')
  else:
   pkg={'name':'analysis-jest','private':True,'type':'commonjs','scripts':{'test':'jest'},'devDependencies':{'jest':'30.2.0'}}
   source='function value(n) { if(n>0) return 1; return 0; }\nfunction caller() { return value(1); }\nfunction uncovered() { return 1; }\nmodule.exports={value,caller,uncovered};\n';path='src/value.cjs'
   test='const {caller}=require("../src/value.cjs");\ntest("value",()=>expect(caller()).toBeGreaterThan(0));\n'
   write('jest.config.cjs','module.exports={testMatch:["**/test/*.test.cjs"],collectCoverageFrom:["src/*.cjs"]};\n')
  write('package.json',json.dumps(pkg));write('tsconfig.json',json.dumps({'compilerOptions':compiler,'include':['src/**/*','test/**/*']}));write(path,source);write('test/value.test.'+('cjs' if case=='jest' else 'ts'),test)
  git('init','-q');git('config','user.name','Framework fixture');git('config','user.email','fixture@example.invalid');git('add','.');git('commit','-qm','before')
  write(path,source.replace('return 1','return 2'));git('add','.');git('commit','-qm','after')
  # Original dependencies remain ignored and snapshots prepare their own links.
  original=git('status','--porcelain')
  raw=command([cli,'project','analyze',root,'--resolve','--operations','impact,coverage','--json','--compact','-o',out/(case+'.html')]);(out/(case+'.json')).write_text(raw);report=json.loads(raw)
  assert report['execution']['status']=='SUCCEEDED',report
  methods=report['matching']['datasets'][0]['methods']
  assert any(m['path']==path and m['status'] in ('EXECUTED','PARTIALLY_EXECUTED') for m in methods),methods
  assert any('uncovered' in m['signature'] and m['status']=='NOT_EXECUTED' for m in methods),methods
  assert git('status','--porcelain')==original
  print(case,'passed',flush=True)
