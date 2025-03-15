#!/bin/sh

echo "BOOK 1"
./jaflToHtml-linux -c -s -o book1.html book1
echo "Done"
echo "Converting to pdf..."
chromium --headless --print-to-pdf="Book 1.pdf" ./book1.html

echo "BOOK 2"
./jaflToHtml-linux -c -s -o book2.html book2
echo ""
echo "Converting to pdf..."
chromium --headless --print-to-pdf="Book 2.pdf" ./book2.html

echo "BOOK 3"
./jaflToHtml-linux -c -s -o book3.html book3
echo ""
echo "Converting to pdf..."
chromium --headless --print-to-pdf="Book 3.pdf" ./book3.html

echo "BOOK 4"
./jaflToHtml-linux -c -s -o book4.html book4
echo ""
echo "Converting to pdf..."
chromium --headless --print-to-pdf="Book 4.pdf" ./book4.html

echo "BOOK 5"
./jaflToHtml-linux -c -s -o book5.html book5
echo ""
echo "Converting to pdf..."
chromium --headless --print-to-pdf="Book 5.pdf" ./book5.html

echo "BOOK 6"
./jaflToHtml-linux -c -s -o book6.html book6
echo ""
echo "Converting to pdf..."
chromium --headless --print-to-pdf="Book 6.pdf" ./book6.html
