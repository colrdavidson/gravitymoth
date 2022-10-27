rm -rf public
mkdir public

# Setup top-level
cp CNAME public/.
cp index.html public/.
cp index.js public/.
cp style.css public/.
cp -r media public/.

# Setup blog
cd ./blog
./build.sh
./gen_site
cp -r docs ../public/blog
cd -

# Setup spall
cp -r spall public/spall
