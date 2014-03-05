#/bin/sh

cd static/img

mkdir -p croped

for img in `ls *.jpg`
do
	convert $img -crop x275+0+0 +repage "croped/$img"
done

rm *.jpg