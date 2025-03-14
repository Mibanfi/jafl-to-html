# jafl-to-html
A tool to convert Java Fabled Lands book into html format and, from there, pdf or epub.

## Downloading
Clone this repository, or download the zip file from Releases.

## Usage
### Preparing Java Fabled Lands
- Download Java Fabled Lands from [here](https://flapp.sourceforge.net/)
- Inside, you will find several zip files, one for each book. They are called 'book1.zip', 'book2.zip' and so on.
- Extract the book files. Your unzipping software will probably put each of them into a new folder automatically, which is exactly what we want. You should have several folders called 'book1', 'book2' and so on.
- Make sure that these folders don't have additional sub-folders nested in them. I.e., the files should be directly inside the folders.
### Running the program
- Download the program.
- Open the command line in the program's directory.
- Run the program. Pass as an argument the address of the book's directory.
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
### What about the character sheet?
- The character sheet (plus a bunch of other goodies) can be downloaded from the [publisher's official website](http://www.sparkfurnace.com/fabled-lands/fl-extras/).

## Optional features
### Adding a cover image
- Find a cover image that you like online.
- Download it and save it as "Cover.jpg" inside the book folder.
- Run the program as before, but add the flag *-c* before you pass any arguments.
- Save the output to pdf as per the steps above.
### Specifying a different name/place to save the output
- Run the program as normal, but pass the flag *-o* immediately followed by the address where you'd like the output to be saved.

## Building from source
The source is a single file written in golang. You can build it like normal (if you've used golang, you know how to do it).

There's also a CSS file containing various styling rules. It is necessary for the result to be properly formatted.
