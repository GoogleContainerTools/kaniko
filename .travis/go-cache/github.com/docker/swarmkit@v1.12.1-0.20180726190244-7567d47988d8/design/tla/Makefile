WORKERS := 4

TLA := docker run --rm -it --workdir /mnt -v ${PWD}:/mnt talex5/tla

.PHONY: all check pdfs tlaps

all: check pdfs tlaps

# Run the TLC model checker
check:
	${TLA} tlc -workers ${WORKERS} SwarmKit.tla -config models/SwarmKit.cfg
	${TLA} tlc -workers ${WORKERS} WorkerImpl.tla -config models/WorkerImpl.cfg

# Run the TLAPS proof checker
tlaps:
	${TLA} tlapm -I /usr/local/lib/tlaps SwarmKit.tla
	${TLA} tlapm -I /usr/local/lib/tlaps WorkerImpl.tla

# Generate a PDF file from a .tla file
%.pdf: %.tla
	[ -d metadir ] || mkdir metadir
	${TLA} java tla2tex.TLA -shade -latexCommand pdflatex -latexOutputExt pdf -metadir metadir $<

pdfs: SwarmKit.pdf Types.pdf Tasks.pdf WorkerSpec.pdf EventCounter.pdf
