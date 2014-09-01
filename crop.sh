#/bin/sh
mkdir -p static/img/croped

cd static/img

for img in `ls *.jpg`
do
	convert $img -crop x275+0+0 +repage "croped/$img"
done

rm *.jpg