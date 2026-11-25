import base64
h=open('web/index.html').read()
css=open('web/app.css').read(); js=open('web/app.js').read(); demo=open('preview/demo.js').read()
fl='data:image/svg+xml;base64,'+base64.b64encode(open('web/flame.svg','rb').read()).decode()
mk='data:image/png;base64,'+base64.b64encode(open('web/mark.png','rb').read()).decode()
h=h.replace('<link rel="stylesheet" href="app.css">','<style>'+css+'</style>')
h=h.replace('<script src="app.js"></script>','<script>'+demo+'</script><script>'+js+'</script>')
h=h.replace('src="flame.svg"','src="'+fl+'"').replace('href="mark.png"','href="'+mk+'"')
open('preview/index.html','w').write(h)
