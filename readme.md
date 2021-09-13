# Introduction

This is a [Gazelle] extension for [OBazl], generating [Bazel] build files for OCaml projects.
It uses [codept] to compute the module dependencies.

# Usage

Okapi configures most of Gazelle's boilerplate with a few helper macros for `WORKSPACE.bazel` and `BUILD.bazel`.

## `WORKSPACE.bazel`

The file `WORKSPACE.bazel` specifies dependencies on Okapi, Gazelle and OBazl and handles project-wide setup:

```bzl
workspace(name = "obazl-project-1")

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Fill in commit and checksum
http_archive(
    name = "okapi",
    strip_prefix = "okapi-<commit>",
    urls = ["https://github.com/tweag/okapi/archive/<commit>.tar.gz"],
    sha256 = "<sha>",
)

# This adds Gazelle and OBazl dependencies as well
load("@okapi//bzl:deps.bzl", "okapi_deps")
okapi_deps()

# Configure default toolchains
load("@okapi//bzl:setup.bzl", "okapi_setup")
okapi_setup(
   ## nix_support = False  ## uncomment if not using nix
)

# This is the standard OBazl setup, requires an existing OPAM repository with the switch and compiler specified here
load("@obazl_rules_ocaml//ocaml:providers.bzl", "BuildConfig", "OpamConfig")
opam = OpamConfig(
    version = "2.0",
    builds = {
        "4.10": BuildConfig(default = True, switch = "4.10", compiler = "4.10", packages = { "ocaml": [] }),
    },
)

load("@obazl_rules_ocaml//ocaml:bootstrap.bzl", "ocaml_configure")
ocaml_configure(build = "4.10", opam = opam)
```

If you want to configure the Go toolchain manually, replace `okapi_setup()` with your own logic, like:

```bzl
load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")

go_rules_dependencies()

go_register_toolchains(version = "1.17")
```

## `BUILD.bazel`

The file `BUILD.bazel` defines the target that integrates Gazelle, so that build file generation can be triggered by
running `bazel run //:gazelle`:

```bzl
load(
    "@bazel_gazelle//:def.bzl",
    "DEFAULT_LANGUAGES",
    "gazelle",
    "gazelle_binary",
)

gazelle_binary(
    name = "gazelle_binary",
    languages = DEFAULT_LANGUAGES + ["@okapi//lang"],
)

gazelle(
    name = "gazelle",
    gazelle = "//:gazelle_binary",
)
```

Okapi provides a convenience macro for this boilerplate.
You can replace the above with:

```bzl
load("@okapi//bzl:generate.bzl", "generate")

generate()
```

Now build files for directories containing OCaml sources will be generated when running:

```sh
bazel run //:gazelle
```

## examples

This repository contains an example project in `test/project-1`. It
has the following OPAM dependencies, which you can install by e.g.
`opam install ipaddr`. Note that you must run `bazel clean --expunge`
after installing OPAM packages.

* angstrom
* ipaddr

Build generation can be observed in action by running the following command in that directory:

```sh
rm -f a/BUILD.bazel a/sub/BUILD.bazel && bazel run //:gazelle && bazel build //a:#A
```

This creates the following `a/BUILD.bazel`:

```bzl
load("@obazl_rules_ocaml//ocaml:rules.bzl", "ocaml_module", "ocaml_ns_library", "ocaml_signature")

ocaml_module(
    name = "a2",
    sig = ":a2_sig",
    struct = ":a2.ml",
    deps = [":f1"],
)

ocaml_signature(
    name = "a2_sig",
    src = ":a2.mli",
    deps = [":f1"],
)

ocaml_module(
    name = "f1",
    sig = ":f1_sig",
    struct = ":f1.ml",
    deps = [],
)

ocaml_signature(
    name = "f1_sig",
    src = ":f1.mli",
    deps = [],
)

ocaml_module(
    name = "a3",
    struct = ":a3.ml",
    deps = [
        ":a2",
        ":f1",
    ],
)

ocaml_ns_library(
    name = "#A",
    submodules = [
        ":a3",
        ":a2",
        ":f1",
    ],
    visibility = ["//visibility:public"],
)
```

# Dune Conversion

If a source directory has no Bazel config, but there is a `dune` file present, the Dune configuration will be used to
populate the attributes `opts` (from `flags`) and `deps_opam` (from `libraries`).

`select` stanzas are parsed in order to find the correct module file names for the library, but the selection of the
correct source file has to be done manually, since there is no (easy) way to check for the presence of dependencies.

Preprocessors are supported as well, causing the addition of a `ppx_executable`, which is then referenced by the
library's modules, using the rules `ppx_module` and `ppx_ns_library`.

Virtual modules are supported.

## Example

Given a Dune config like this:

```dune
(library
 (name sub_lib)
 (public_name sub-lib)
 (flags (:standard -open Angstrom))
 (libraries
   angstrom
   re
   ipaddr
   (select final.ml from
     (angstrom -> choice1.ml)
     (-> choice2.ml))
  ))

(library
 (name sub_extra_lib)
 (public_name sub-extra-lib)
 (preprocess (pps ppx_inline_test))
 (modules foo bar))
```

The generated build will be:

```dune
load("@obazl_rules_ocaml//ocaml:rules.bzl", "ocaml_module", "ocaml_ns_library", "ocaml_signature")

ocaml_module(
    name = "final",
    deps_opam = [
        "angstrom",
        "re",
        "ipaddr",
    ],
    opts = [
        "-open",
        "Angstrom",
    ],
    struct = ":final.ml",
)

ocaml_module(
    name = "sub",
    deps_opam = [
        "angstrom",
        "re",
        "ipaddr",
    ],
    opts = [
        "-open",
        "Angstrom",
    ],
    struct = ":sub.ml",
)

# okapi:auto
ocaml_ns_library(
    name = "#Sub_lib",
    submodules = [
        "final",
        "sub",
    ],
    visibility = ["//visibility:public"],
)

ppx_executable(
    name = "ppx_sub_extra_lib",
    deps_opam = ["ppx_inline_test"],
    main = "@obazl_rules_ocaml//dsl:ppx_driver",
)

ppx_module(
    name = "foo",
    deps_opam = [],
    opts = [],
    ppx = ":ppx_sub_extra_lib",
    ppx_print = "@ppx//print:text",
    struct = ":foo.ml",
)

ppx_module(
    name = "bar",
    deps_opam = [],
    opts = [],
    ppx = ":ppx_sub_extra_lib",
    ppx_print = "@ppx//print:text",
    struct = ":bar.ml",
)

ppx_ns_library(
    name = "#Sub_extra_lib",
    submodules = [
        "foo",
        "bar",
    ],
    visibility = ["//visibility:public"],
)
```

# Multilib Builds

If a build file defines more than one library, as is also possible with Dune, the generator cannot decide which library
should become the owner of a newly added module when updating.

The user may therefore mark one of the libraries as the one that owns all new files by placing a comment right before
the library target:

```bzl
# okapi:auto
ocaml_ns_library(
    name = "#A",
    submodules = [...]
)
```

Libraries converted from a Dune config are automatically annotated with this comment if they don't have an explicit
module list.

# Local Dune Dependencies

Dune allows the `libraries` stanza to be a mix of OPAM dependencies and libraries defined in the current project.
For Bazel, the two need to be differentiated.
For this purpose, the Gazelle resolver step examines the depspecs in `libraries` and looks for a matching library rule,
adding it to the `deps` attribute on success and `deps_opam` otherwise.
Valid matches are the values form both the `name` and `public_name` stanzas in the Dune config.

This feature can be used manually as well, in which case either the library rule's `name` attribute is matched, or, if
available, the `public_name` from a comment:

```bzl
# okapi:public_name acme.missiles
ocaml_ns_library(
    name = "#Acme_missiles",
    submodules = [...]
)
```

This would only be relevant when using a mix of Dune and automatic builds.

# Tests

The project contains basic Go unit tests as well as Bazel integration tests.

They can be executed, respectively, with:

```sh
$ bazel test '//lang:*'
$ bazel test '//test/...'
```

[Gazelle]: https://github.com/bazelbuild/bazel-gazelle
[OBazl]: https://github.com/obazl/rules_ocaml
[Bazel]: https://bazel.build
[codept]: https://github.com/Octachron/codept
