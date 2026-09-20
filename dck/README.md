# DCK version

This directory contains the construction-kit version of bilizir-demo. The original Go sources are preserved at their original paths (revision `4720c97e03f3d2392211a37755d41b752fbad288`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/bilizir-demo` and this version with `go run ./dck/cmd/bilizir-demo` from the repository root.

The choreography and assets stay local; reusable rendering and effects live in `../../lib/democonstructionkit`. Second Reality retains its original ST3 music synchronization.
