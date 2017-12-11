MERMAID=mmdc

all: 	../images/test.mermaid.png

../images/test.mermaid.png:
	$(MERMAID) -i test.mermaid -w 500 -o ../images/test.mermaid.svg

clean:
		cd ..; rm -f `find . -name '*.mermaid.png'`;
