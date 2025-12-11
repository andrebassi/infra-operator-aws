#!/bin/bash
# Script para validar documentação Go seguindo padrões oficiais
# Uso: ./scripts/validate-docs.sh

set -e

BASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "================================================"
echo "VALIDANDO DOCUMENTAÇÃO GO"
echo "================================================"
echo ""

total_files=0
missing_pkg_comment=0

# Função para verificar package comment
check_package_comment() {
    local file="$1"
    if ! head -n 5 "$file" | grep -q "^// Package"; then
        echo "⚠ Package comment faltando: $(basename $file)"
        ((missing_pkg_comment++))
        return 1
    fi
    return 0
}

# Validar principais diretórios
for dir in api/v1alpha1 controllers internal/ports internal/domain internal/usecases internal/adapters/aws pkg/mapper pkg/clients; do
    [ ! -d "$BASE_DIR/$dir" ] && continue
    echo "Validando $dir/..."
    
    find "$BASE_DIR/$dir" -name "*.go" ! -name "*_test.go" ! -name "zz_generated*" | while read -r file; do
        ((total_files++))
        check_package_comment "$file" || true
    done
done

echo ""
echo "================================================"
echo "RESULTADO"
echo "================================================"
echo "Total de arquivos validados: $total_files"

if [ $missing_pkg_comment -eq 0 ]; then
    echo "✅ TODOS OS ARQUIVOS ESTÃO COM PACKAGE COMMENTS!"
    exit 0
else
    echo "⚠ $missing_pkg_comment package comments faltando"
    exit 1
fi
