# notes 
```bash
# 1. Configure the build and set the installation prefix
cmake -B build \
  -DCMAKE_INSTALL_PREFIX="$HOME/install" \
  -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_INSTALL_RPATH="@loader_path/../lib" \
  -DCMAKE_BUILD_WITH_INSTALL_RPATH=ON

# 2. Compile the project
cmake --build build --parallel

# 3. Install the project to $HOME/install
cmake --install build
```