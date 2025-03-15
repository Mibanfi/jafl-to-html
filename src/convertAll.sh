#!/bin/sh

echo "BOOK 1"
./jaflToHtml-linux -c -s book1
echo ""
mv output.html book1.html

echo "BOOK 2"
./jaflToHtml-linux -c -s book2
echo ""
mv output.html book2.html

echo "BOOK 3"
./jaflToHtml-linux -c -s book3
echo ""
mv output.html book3.html

echo "BOOK 4"
./jaflToHtml-linux -c -s book4
echo ""
mv output.html book4.html

echo "BOOK 5"
./jaflToHtml-linux -c -s book5
echo ""
mv output.html book5.html

echo "BOOK 6"
./jaflToHtml-linux -c -s book6
echo ""
mv output.html book6.html
