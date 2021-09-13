package okapi

import (
  "fmt"
  "log"
  "sort"
  "strings"

  "github.com/bazelbuild/bazel-gazelle/rule"
)

type KeyValue struct {
  key string
  value string
}

type ModuleAlt struct {
  cond string
  choice string
}

type ModuleChoice struct {
  out string
  alts []ModuleAlt
}

func ppxName(libName string) string { return "ppx_" + libName }

func addAttrs(slug string, r *rule.Rule, kind PpxKind) {
  if ppx, isDirect := kind.(PpxDirect); isDirect {
    r.SetAttr("ppx", ":" + ppxName(slug))
    r.SetAttr("ppx_print", "@ppx//print:text")
    if contains("ppx_inline_test", ppx.deps) { r.SetAttr("ppx_tags", []string{"inline-test"}) }
  }
}

func extraRules(kind PpxKind, slug string) []RuleResult {
  if ppx, isDirect := kind.(PpxDirect); isDirect { return ppx.exe(slug) }
  return nil
}

func libSuffix(library bool) string {
  if library { return "library" } else { return "archive" }
}

type LibraryKind interface {
  ruleKind(library bool) string
  ppx() bool
  wrapped() bool
}

type LibNsPpx struct {}
type LibNs struct {}
type LibPpx struct {}
type LibPlain struct {}

func (LibNsPpx) ruleKind(library bool) string { return "ppx_ns_" + libSuffix(library) }
func (LibNs) ruleKind(library bool) string { return "ocaml_ns_" + libSuffix(library) }
func (LibPpx) ruleKind(library bool) string { return "ppx_" + libSuffix(library) }
func (LibPlain) ruleKind(library bool) string { return "ocaml_" + libSuffix(library) }

func (LibNsPpx) ppx() bool { return true }
func (LibNs) ppx() bool { return false }
func (LibPpx) ppx() bool { return true }
func (LibPlain) ppx() bool { return false }

func (LibNsPpx) wrapped() bool { return true }
func (LibNs) wrapped() bool { return true }
func (LibPpx) wrapped() bool { return false }
func (LibPlain) wrapped() bool { return false }

type ExeKind interface {
  ruleKind(library bool) string
  ppx() bool
}

type ExePpx struct {}
type ExePlain struct {}

func (ExePpx) ruleKind(library bool) string { return "ppx_executable" }
func (ExePlain) ruleKind(library bool) string { return "ocaml_executable" }

func (ExePpx) ppx() bool { return true }
func (ExePlain) ppx() bool { return false }

type ComponentKind interface {
  componentRule(component Component, library bool) *rule.Rule
  extraDeps() []string
}

type Library struct {
  virtualModules []Source
  implements string
  kind LibraryKind
}

type Executable struct {
  kind ExeKind
}

// TODO store stuff like auto, exclude in annotations
type Component struct {
  core ComponentCore
  modules []Source
  depsOpam []string
  ppx PpxKind
  kind ComponentKind
  annotations []string
}

func moduleAttr(wrapped bool) string {
  if wrapped { return "submodules" } else { return "modules" }
}

func nsName(name string) string {
  return "#" + strings.Title(strings.ReplaceAll(name, "-", "_"))
}

func targetNames(deps []string) []string {
  var result []string
  for _, dep := range deps { result = append(result, ":" + dep) }
  sort.Strings(result)
  return result
}

func libraryModules(srcs []Source) []string {
  var result []string
  for _, src := range srcs {
    if src.generator.libraryModule() {
      result = append(result, ":" + src.name)
    }
  }
  sort.Strings(result)
  return result
}

func (lib Library) componentRule(component Component, library bool) *rule.Rule {
  libName := "lib-" + component.core.name
  if lib.kind.wrapped() { libName = nsName(component.core.name) }
  r := rule.NewRule(lib.kind.ruleKind(library), libName)
  mods := append(component.modules, lib.virtualModules...)
  r.SetAttr(moduleAttr(lib.kind.wrapped()), libraryModules(mods))
  if lib.implements != "" {
    r.AddComment("# okapi:implements " + lib.implements)
    r.AddComment("# okapi:implementation " + component.core.publicName)
  }
  return r
}

func (exe Executable) componentRule(component Component, library bool) *rule.Rule {
  r := rule.NewRule(exe.kind.ruleKind(library), "exe-" + component.core.publicName)
  r.SetAttr("main", component.core.name)
  r.SetAttr("deps", libraryModules(component.modules))
  return r
}

func (lib Library) extraDeps() []string {
  if lib.implements == "" { return nil } else { return []string{lib.implements} }
}

func (Executable) extraDeps() []string { return nil }

type RuleResult struct {
  rule *rule.Rule
  deps []string
}

func extendAttr(r *rule.Rule, attr string, vs []string) {
  if len(vs) > 0 { r.SetAttr(attr, append(r.AttrStrings(attr), vs...)) }
}

func appendAttr(r *rule.Rule, attr string, v string) {
  r.SetAttr(attr, append(r.AttrStrings(attr), v))
}

func commonAttrs(component Component, r *rule.Rule, deps []string) RuleResult {
  libDeps := append(append(component.depsOpam, component.ppx.depsOpam()...), component.kind.extraDeps()...)
  extendAttr(r, "opts", component.core.flags)
  if len(deps) > 0 { r.SetAttr("deps", targetNames(deps)) }
  addAttrs(component.core.name, r, component.ppx)
  return RuleResult{r, libDeps}
}

func sigTarget(src Source) string { return src.name + "__sig" }

func signatureRule(component Component, src Source, deps []string) RuleResult {
  r := rule.NewRule("ocaml_signature", sigTarget(src))
  r.SetAttr("src", ":" + src.name + ".mli")
  return commonAttrs(component, r, deps)
}

func virtualSignatureRule(libName string, src Source) *rule.Rule {
  r := rule.NewRule("ocaml_signature", src.name)
  r.SetAttr("src", ":" + src.name + ".mli")
  r.AddComment(fmt.Sprintf("# okapi:virt %s", libName))
  return r
}

func moduleRuleName(component Component) string {
  if component.ppx.isPpx() { return "ppx_module" } else { return "ocaml_module" }
}

func moduleRule(component Component, src Source, struct_ string, deps []string) RuleResult {
  r := rule.NewRule(moduleRuleName(component), src.name)
  r.SetAttr("struct", struct_)
  if src.intf {
    r.SetAttr("sig", ":" + sigTarget(src))
  } else if lib, isLib := component.kind.(Library); isLib && lib.implements != "" {
    r.AddComment(fmt.Sprintf("# okapi:implements %s", lib.implements))
  }
  return commonAttrs(component, r, deps)
}

func defaultModuleRule(component Component, src Source, deps []string) RuleResult {
  return moduleRule(component, src, ":" + src.name + ".ml", deps)
}

func lexRules(component Component, src Source, deps []string) []RuleResult {
  structName := src.name + "_ml"
  lexRule := rule.NewRule("ocaml_lex", structName)
  lexRule.SetAttr("src", ":" + src.name + ".mll")
  lexRule.SetAttr("out", ":" + src.name + ".ml")
  modRule := moduleRule(component, src, ":" + structName, deps)
  modRule.rule.SetAttr("opts", []string{"-w", "-39"})
  return []RuleResult{{lexRule, nil}, modRule}
}

func remove(name string, deps []string) []string {
  var result []string
  for _, dep := range deps {
    if dep != name { result = append(result, dep) }
  }
  return result
}

func librarySourceRules(component Component, lib Library, sources Deps) []RuleResult {
  var rules []RuleResult
  var m SourceSlice = lib.virtualModules
  m.Sort()
  for _, src := range m {
    if src.generator == nil {
      log.Fatalf("no generator for %#v", src)
    }
    cleanDeps := remove(src.name, src.deps)
    rules = append(rules, commonAttrs(component, virtualSignatureRule(component.core.publicName, src), cleanDeps))
  }
  return rules
}

// If the source was generated, the module rule will be handled by the generator logic.
// This still uses a potential interface though, since that may be supplied unmanaged.
func sourceRule(src Source, component Component) []RuleResult {
  var rules []RuleResult
  cleanDeps := remove(src.name, src.deps)
  if src.intf { rules = append(rules, signatureRule(component, src, cleanDeps)) }
  if _, isNoGen := src.generator.(NoGenerator); isNoGen {
    rules = append(rules, defaultModuleRule(component, src, cleanDeps))
  } else if _, isLexer := src.generator.(Lexer); isLexer {
    rules = append(rules, lexRules(component, src, cleanDeps)...)
  } else if _, isChoice := src.generator.(Choice); isChoice {
    rules = append(rules, defaultModuleRule(component, src, cleanDeps))
  } else {
    log.Fatalf("no generator for %#v", src)
  }
  return rules
}

func sourceRules(sources Deps, component Component) []RuleResult {
  var rules []RuleResult
  rules = append(rules, extraRules(component.ppx, component.core.name)...)
  var m SourceSlice = component.modules
  m.Sort()
  for _, src := range m {
    rules = append(rules, sourceRule(src, component)...)
  }
  if lib, isLib := component.kind.(Library); isLib {
    rules = append(rules, librarySourceRules(component, lib, sources)...)
  }
  return rules
}

func setLibraryModules(component Component, r *rule.Rule) {
}

func componentRule(component Component, library bool) RuleResult {
  r := component.kind.componentRule(component, library)
  if component.core.auto { r.AddComment("# okapi:auto") }
  r.AddComment("# okapi:public_name " + component.core.publicName)
  r.SetAttr("visibility", []string{"//visibility:public"})
  return RuleResult{r, component.depsOpam}
}

func component(sources Deps, component Component, library bool) []RuleResult {
  return append(sourceRules(sources, component), componentRule(component, library))
}

type ComponentSources struct {
  component ComponentSpec
  sources []Source
}

func componentWithSources(comp ComponentSpec, generated map[string][]string, sources Deps) ComponentSources {
  return ComponentSources{
    component: comp,
    sources: moduleSources(append(comp.modules.names(), generated[comp.core.name]...), sources, comp.choices),
  }
}

func componentsWithSources(components []ComponentSpec, generated map[string][]string, sources Deps) []ComponentSources {
  var result []ComponentSources
  for _, comp := range components { result = append(result, componentWithSources(comp, generated, sources)) }
  return result
}

func filterAuto(auto []Source, spec ModuleSpec) []Source {
  if _, isAuto := spec.(AutoModules); isAuto {
    return auto
  } else if exclude, isExclude := spec.(ExcludeModules); isExclude {
    var result []Source
    for _, src := range auto {
      found := false
      for _, ex := range exclude.modules {
        if ex == src.name { found = true }
      }
      if !found { result = append(result, src) }
    }
    return result
  } else {
    return nil
  }
}

func specComponent(comp ComponentSources, sources Deps, auto []Source) Component {
  modules := comp.sources
  modules = append(modules, filterAuto(auto, comp.component.modules)...)
  return Component{
    core: comp.component.core,
    modules: modules,
    depsOpam: comp.component.depsOpam,
    ppx: comp.component.ppx,
    kind: comp.component.kind.toObazl(comp.component.ppx, sources),
    annotations: nil,
  }
}

func specComponents(spec PackageSpec, sources Deps) []Component {
  generated := assignGenerated(spec)
  withSources := componentsWithSources(spec.components, generated, sources)
  auto := autoModules(withSources, sources)
  var result []Component
  for _, comp := range withSources { result = append(result, specComponent(comp, sources, auto)) }
  return result
}

// Update an existing build that has been manually amended by the user to contain more than one library.
// In that case, all submodule assignments are static, and only the module/signature rules are updated.
// TODO when `select` directives are used from dune, they don't create module rules for the choices.
// When gazelle is then run in update mode, they will be created.
// Either check for rules that select one of the choices or add exclude rules in comments.
func multilib(spec PackageSpec, sources Deps, library bool) []RuleResult {
  components := specComponents(spec, sources)
  var rules []RuleResult
  for _, comp := range components {
    rules = append(rules, component(sources, comp, library)...)
  }
  return rules
}

func libChoices(libs []ComponentSources) map[string]bool {
  result := make(map[string]bool)
  for _, lib := range libs {
    for _, mod := range lib.sources {
      if c, isChoice := mod.generator.(Choice); isChoice {
        for _, a := range c.alts {
          result[depName(a.choice)] = true
        }
      }
    }
  }
  return result
}

func autoModules(components []ComponentSources, sources Deps) []Source {
  knownModules := make(map[string]bool)
  choices := libChoices(components)
  var auto []Source
  for _, component := range components {
    for _, mod := range component.sources { knownModules[mod.name] = true }
    if lib, isLib := component.component.kind.(LibSpec); isLib {
      for _, mod := range lib.virtualModules { knownModules[mod] = true }
    }
  }
  for name, src := range sources {
    if _, exists := knownModules[name]; !exists {
      if _, isChoice := choices[name]; !isChoice {
        auto = append(auto, src)
      }
    }
  }
  return auto
}
