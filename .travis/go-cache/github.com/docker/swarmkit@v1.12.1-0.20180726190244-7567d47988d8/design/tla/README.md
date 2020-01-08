This directory contains documentation for SwarmKit using [TLA+][] notation.

Run `make pdfs` to render these documents as PDF files.
The best one to start with is `SwarmKit.pdf`, which introduces the TLA+ notation
and describes the overall components of SwarmKit.

The specifications can also be executed by the TLC model checker to help find
mistakes. Use `make check` to run the checks.

If you want to edit these specifications, you will probably want to use the [TLA+ Toolbox][],
which provides a GUI.

[TLA+]: https://en.wikipedia.org/wiki/TLA%2B
[TLA+ Toolbox]: http://lamport.azurewebsites.net/tla/toolbox.html
