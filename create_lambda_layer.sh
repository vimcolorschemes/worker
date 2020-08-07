PY_DIR="build/python/lib/python3.7/site-packages"

echo "Create $PY_DIR"
mkdir -p $PY_DIR

echo "Create virtualenv"
python3 -m venv env
source env/bin/activate

echo "Compile requirements"
pip install -r requirements.in --no-deps -t $PY_DIR

echo "Zip build"
cd build
zip -r ../colorschemes_dev-worker-layer.zip .
cd ..
rm -r build
