package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

type html struct {
	Body body `xml:"body"`
}
type body struct {
	Content string `xml:",innerxml"`
}

func main() {
	b := []byte(`<!DOCTYPE html>
<html>
    <head>
        <title>
            Title of the document
        </title>
    </head>
    <body>
        body content 
        <p>more content</p>
                                                <script>^M
                                                (function() {^M
                                                    var s = "_" + Math.random().toString(36).slice(2);^M
                                                    document.write('<div id="' + s + '"></div>');^M
                                                    (window.slotbydup=window.slotbydup || []).push({^M
                                                        id: '3011945',^M
                                                        container: s,^M
                                                        size: '260,90',^M
                                                        display: 'inlay-fix'^M
                                                    });^M
                                                })();^M
                                                </script> ^M
    </body>

</html>`)

	h := html{}
	err := xml.NewDecoder(bytes.NewBuffer(b)).Decode(&h)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	fmt.Println(h.Body.Content)
}
