#!/usr/bin/env bash
#
# install.sh — 安装增量分支驱动项目训练 Skill 到 Codex/Claude Code
#
# 用法:
#   ./install.sh                          # 安装到 ~/.codex/skills/
#   ./install.sh --target ~/.claude/skills  # 安装到自定义目录
#   ./install.sh --dry-run                # 预览操作，不实际执行
#   ./install.sh --force                  # 跳过确认，直接安装
#
# 功能:
#   - 复制 Skill 目录到目标位置
#   - 已有版本自动备份（带时间戳）
#   - 验证目录结构完整性
#   - 输出安装报告

set -euo pipefail

# —— 配置 ——————————————————————————————————————————————

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(dirname "$SCRIPT_DIR")"  # incremental-branch-training/
SKILL_NAME="$(basename "$SKILL_DIR")"

DEFAULT_TARGET="${HOME}/.codex/skills"
TARGET_DIR="${DEFAULT_TARGET}"
DRY_RUN=false
FORCE=false

# —— 颜色 ——————————————————————————————————————————————

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# —— 工具函数 ——————————————————————————————————————————————

msg()  { echo -e "${BLUE}[install]${NC} $*"; }
ok()   { echo -e "${GREEN}[✓]${NC} $*"; }
warn() { echo -e "${YELLOW}[!]${NC} $*"; }
err()  { echo -e "${RED}[✗]${NC} $*"; }

separator() {
    echo "————————————————————————————————————————————"
}

# —— 参数解析 ——————————————————————————————————————————————

while [[ $# -gt 0 ]]; do
    case "$1" in
        --target)
            TARGET_DIR="${2}"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --help|-h)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --target DIR    安装目标目录 (默认: ~/.codex/skills)"
            echo "  --dry-run       预览操作，不实际执行"
            echo "  --force         跳过确认提示，直接安装"
            echo "  --help, -h      显示帮助信息"
            exit 0
            ;;
        *)
            err "未知参数: $1"
            echo "使用 --help 查看用法。"
            exit 1
            ;;
    esac
done

INSTALL_PATH="${TARGET_DIR}/${SKILL_NAME}"

# —— 入口 ——————————————————————————————————————————————

echo ""
echo -e "${BOLD}增量分支驱动项目训练 — Skill 安装${NC}"
separator
msg "源目录:   ${SKILL_DIR}"
msg "目标位置: ${INSTALL_PATH}"
if ${DRY_RUN}; then
    msg "模式:     DRY-RUN（预览，不实际执行）"
fi
echo ""

# —— 检查源目录 ——————————————————————————————————————————————

REQUIRED_FILES=(
    "SKILL.md"
    "agents/openai.yaml"
    "scripts/init_training_docs.py"
    "references/stage-workflow.md"
    "references/design-review.md"
    "references/code-control.md"
    "references/deliverable-templates.md"
)

MISSING_FILES=()
for f in "${REQUIRED_FILES[@]}"; do
    if [[ ! -f "${SKILL_DIR}/${f}" ]]; then
        MISSING_FILES+=("$f")
    fi
done

if [[ ${#MISSING_FILES[@]} -gt 0 ]]; then
    err "源目录缺少必需文件:"
    for f in "${MISSING_FILES[@]}"; do
        echo "    - $f"
    done
    echo ""
    err "安装中止。请确认你在 Skill 仓库根目录执行此脚本。"
    exit 1
fi
ok "源文件完整性检查通过"

# —— 检查目标目录 ——————————————————————————————————————————————

if ! ${DRY_RUN}; then
    mkdir -p "${TARGET_DIR}"
fi

# —— 备份已有版本 ——————————————————————————————————————————————

if [[ -d "${INSTALL_PATH}" ]]; then
    BACKUP_DIR="${TARGET_DIR}/${SKILL_NAME}.bak.$(date +%Y%m%d_%H%M%S)"
    warn "目标位置已存在: ${INSTALL_PATH}"

    if ${DRY_RUN}; then
        msg "[DRY-RUN] 将备份到: ${BACKUP_DIR}"
    else
        if ${FORCE}; then
            msg "自动备份到: ${BACKUP_DIR}"
        else
            echo ""
            read -r -p "是否备份已有版本并继续安装？[Y/n] " REPLY
            if [[ "${REPLY}" != "Y" && "${REPLY}" != "y" && -n "${REPLY}" ]]; then
                echo ""
                msg "安装取消。"
                exit 0
            fi
            echo ""
        fi
        mv "${INSTALL_PATH}" "${BACKUP_DIR}"
        ok "已有版本已备份到: ${BACKUP_DIR}"
    fi
fi

# —— 复制 Skill 文件 ——————————————————————————————————————————————

if ${DRY_RUN}; then
    msg "[DRY-RUN] 将复制: ${SKILL_DIR} -> ${INSTALL_PATH}"
else
    cp -R "${SKILL_DIR}" "${INSTALL_PATH}"
    ok "Skill 文件已复制"
fi

# —— 验证安装 ——————————————————————————————————————————————

separator
echo ""
msg "验证安装结构..."

INSTALL_OK=true
for f in "${REQUIRED_FILES[@]}"; do
    if [[ -f "${INSTALL_PATH}/${f}" ]]; then
        ok "  ${f}"
    else
        err "  ${f} — 缺失"
        INSTALL_OK=false
    fi
done

# 可选的新文件（不阻塞安装，但会提示）
OPTIONAL_FILES=(
    "references/project-presets.md"
    "references/examples.md"
    "references/scoring-rubrics.md"
    "references/learner-levels.md"
    "references/mock-interview.md"
)

for f in "${OPTIONAL_FILES[@]}"; do
    if [[ -f "${INSTALL_PATH}/${f}" ]]; then
        ok "  ${f}"
    else
        warn "  ${f} — 可选文件缺失（不影响核心功能）"
    fi
done

# —— 安装报告 ——————————————————————————————————————————————

separator
echo ""
if ${DRY_RUN}; then
    echo -e "${BOLD}DRY-RUN 完成 — 以上为预览，未实际执行。${NC}"
    echo ""
    echo "移除 --dry-run 后重新执行以完成安装。"
elif ${INSTALL_OK}; then
    echo -e "${BOLD}安装成功 ✓${NC}"
    echo ""
    echo "Skill 已安装到: ${INSTALL_PATH}"
    echo ""
    echo "使用方式："
    echo ""
    echo "  Codex:"
    echo "    在对话中输入:"
    echo "    \$incremental-branch-training 带我做一个 [项目类型]"
    echo ""
    echo "  Claude Code:"
    echo "    将以下内容粘贴到项目对话中，或放入项目的 CLAUDE.md / AGENTS.md:"
    echo ""
    echo "    — 参见 README.md 中的「Claude Code 使用方式」—"
    echo ""
    echo "快速验证安装:"
    echo "  find ~/.codex/skills/incremental-branch-training -maxdepth 3 -type f | sort"
else
    err "安装验证失败 — 部分文件缺失。"
    echo ""
    echo "请检查源目录是否完整，或重新执行安装。"
    exit 1
fi

exit 0
