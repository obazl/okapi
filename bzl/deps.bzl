load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:utils.bzl", "maybe")

def okapi_deps():
    maybe(
        http_archive,
        "bazel_skylib",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-skylib/releases/download/1.0.2/bazel-skylib-1.0.2.tar.gz",
            "https://github.com/bazelbuild/bazel-skylib/releases/download/1.0.2/bazel-skylib-1.0.2.tar.gz",
        ],
        sha256 = "97e70364e9249702246c0e9444bccdc4b847bed1eb03c5a3ece4f83dfe6abc44",
    )

    ################  OBazl 2 ###############
    maybe(
        git_repository,
        name = "obazl_tools_obazl",
        remote = "https://github.com/obazl/tools_obazl",
        branch = "dev",
    )

    # maybe(
    #     git_repository,
    #     name = "obazl_tools_opam",
    #     remote = "https://github.com/obazl/tools_opam",
    #     branch = "nocamlfind",
    # )

    maybe(
        git_repository,
        name = "obazl_rules_ocaml",
        remote = "https://github.com/obazl/rules_ocaml",
        branch = "nocamlfind",
    )
    # maybe(
    #     http_archive,
    #     name = "obazl_rules_ocaml",
    #     strip_prefix = "rules_ocaml-ad4c5562fb4a3cade5764129d61d4e2f6fdf77ac",
    #     url = "https://github.com/tek/rules_ocaml/archive/ad4c5562fb4a3cade5764129d61d4e2f6fdf77ac.tar.gz",
    #     sha256 = "5df21041e154d3f1e6914779862cf1843d6e7364fdcb72306441a22ced3147b8",
    # )

    ################ TWEAG ################
    maybe(
        http_archive,
        name = "io_tweag_rules_nixpkgs",
        sha256 = "8204bb4db303cc29261548f6cad802f63cddc053f8a747401561e0c36dcd49a8",
        strip_prefix = "rules_nixpkgs-1d29b771db75b8d18f5f5baa8f99be16325b897e",
        urls = ["https://github.com/tweag/rules_nixpkgs/archive/1d29b771db75b8d18f5f5baa8f99be16325b897e.tar.gz"],
    )
    maybe(
        http_archive,
        name = "io_bazel_rules_go",
        sha256 = "69de5c704a05ff37862f7e0f5534d4f479418afc21806c887db544a316f3cb6b",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.27.0/rules_go-v0.27.0.tar.gz",
            "https://github.com/bazelbuild/rules_go/releases/download/v0.27.0/rules_go-v0.27.0.tar.gz",
        ],
    )
    maybe(
        http_archive,
        name = "bazel_gazelle",
        sha256 = "62ca106be173579c0a167deb23358fdfe71ffa1e4cfdddf5582af26520f1c66f",
        urls = [
            "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.23.0/bazel-gazelle-v0.23.0.tar.gz",
            "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.23.0/bazel-gazelle-v0.23.0.tar.gz",
        ],
    )
