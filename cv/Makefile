all: render_en render_nl

render_en:
	pandoc cv_eng.html -c cv.css -t html5 -o francis_begyn_cv_eng.pdf \
		-V margin-top=3 -V margin-left=3 -V margin-right=3 -V margin-bottom=3 \
		-V lettersize=a4

render_nl:
	pandoc cv_nl.html -c cv.css -t html5 -o francis_begyn_cv_nl.pdf \
		-V margin-top=3 -V margin-left=3 -V margin-right=3 -V margin-bottom=3 \
		-V lettersize=a4
