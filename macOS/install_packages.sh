#!/bin/bash

# License
# 本项目受 GNU General Public License v3.0 约束
# 详见 LICENSE 文件或 https://www.gnu.org/licenses/gpl-3.0.html

# 切换 Homebrew 源为中国源
echo "Switching Homebrew source to China..."
brew update-reset
echo 'export HOMEBREW_BREW_GIT_REMOTE="https://mirrors.ustc.edu.cn/brew.git"' >> ~/.bash_profile
echo 'export HOMEBREW_CORE_GIT_REMOTE="https://mirrors.ustc.edu.cn/homebrew-core.git"' >> ~/.bash_profile
source ~/.bash_profile
echo "Homebrew source switched to China."

# 获取脚本所在目录的路径
script_dir=$(dirname "$0")

# 定义软件列表文件路径
software_list="${script_dir}/packages.txt"

# 检查软件列表文件是否存在
if [ ! -f "$software_list" ]; then
    echo "软件列表文件 $software_list 不存在!"
    exit 1
fi

# 从清单中提取软件包 ID，兼容两种格式：
#   - YAML manifest：`- id: xxx` 行
#   - 纯 TXT 列表：每行一个 ID
list_packages() {
    awk '
        /^[[:space:]]*#/ { next }
        /^[[:space:]]*$/ { next }
        /^[[:space:]]*-[[:space:]]*id[[:space:]]*:/ {
            sub(/^[[:space:]]*-[[:space:]]*id[[:space:]]*:[[:space:]]*/, "")
            sub(/[[:space:]]*$/, "")
            print
            next
        }
        /:/ { next }
        { sub(/[[:space:]]*$/, ""); print }
    ' "$software_list"
}

# 逐行读取软件列表文件并安装软件
list_packages | while IFS= read -r package; do
    [ -n "$package" ] || continue
    echo "Checking if $package is installed..."
    if brew list --versions "$package" > /dev/null; then
        echo "$package 已经安装，跳过。"
    else
        echo "Installing $package..."
        brew install "$package"
    fi
done

echo "所有软件安装完成！"
