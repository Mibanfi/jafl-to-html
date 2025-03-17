# jafl-to-html
A tool to convert Java Fabled Lands book into html format and, from there, pdf or epub.

## Usage
- Download Java Fabled Lands from [here](https://flapp.sourceforge.net/)
- Download the program.
- Open the command line in the program's directory.
    - Run the program (you can find executables in *release*). Pass as an argument the address of the book's directory.
    - If you add a second argument, that will be the address of the output file. It should end in .html
    - By default, the program will put all books together in a single file. If you only want to convert *one* book, please pass the flag *-b* followed by the number of the book you would like to convert.
- Presto, it's done!
    - If you move the file around, or delete the book folder, images may not work anymore.
    - Make sure 'jafl.css' is in the same directory as the html file.
### Converting to pdf
- Open the freshly baked html file with your browser of choice.
    - Please make sure you didn't move the html file. This is an inconvenient I need to fix.
    - Please also make sure the file 'jafl.css' is in the same directory as the html file.
- Right click on a part of the page where there are no images or links.
- Select "Print".
    - Alternatively: press Ctrl + P.
- Save it as pdf.
- Ta-da! Now you have a pdf. ***Section links still work!***

## Building from source
The source is a single file written in golang. You can build it like normal (if you've used golang, you know how to do it).

There's also a CSS file containing various styling rules. It is necessary for the result to be properly formatted.
