# load("@obazl_rules_ocaml//ocaml:repositories.bzl", "ocaml_dependencies")
load("@io_tweag_rules_nixpkgs//nixpkgs:nixpkgs.bzl", "nixpkgs_git_repository")
load("@io_tweag_rules_nixpkgs//nixpkgs:toolchains/go.bzl", "nixpkgs_go_configure")
load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains", "go_host_sdk")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

def okapi_setup(nix_support = True, ws = "WORKSPACE.bazel"):
    if nix_support:
        nixpkgs_git_repository(
            name = "nixpkgs",
            revision = "21.05",
        )
        nixpkgs_go_configure(repository = "@nixpkgs", fail_not_supported = False)
    go_rules_dependencies()
    go_host_sdk(name = "go_sdk")
    gazelle_dependencies(go_repository_default_config = "@//:" + ws,
                         go_sdk="go_sdk")

def okapi_setup_legacy():
    okapi_setup(ws = "WORKSPACE")
