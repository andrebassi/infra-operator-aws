#!/usr/bin/env python3
"""
Migrate Mintlify docs to Docusaurus format
"""
import os
import re
import shutil
from pathlib import Path

SOURCE_DIR = Path("/Users/andrebassi/works/infra-operator/docs")
TARGET_DIR = Path("/Users/andrebassi/works/infra-operator/docs/docusaurus/docs")

# Files/dirs to skip
SKIP = {
    'mint.json', 'docusaurus', 'logo', 'favicon.svg', 'reports', 'archive',
    'DOCUMENTATION_ORGANIZATION.md', 'Taskfile.yaml', '.iam-policies.md'
}

# Mapping of source paths to target paths
PATH_MAPPING = {
    'introduction.mdx': 'introduction.md',
    'installation.mdx': 'installation.md',
    'QUICKSTART.md': 'quickstart.md',
    'ARCHITECTURE.md': 'architecture.md',
    'CLEAN_ARCHITECTURE.md': 'advanced/clean-architecture.md',
    'DEVELOPMENT.md': 'advanced/development.md',
    'DEPLOYMENT_GUIDE.md': 'advanced/deployment.md',
    'SERVICES_GUIDE.md': 'guides/services-overview.md',
    'PROMETHEUS_QUERIES.md': 'features/prometheus-queries.md',
}


def clean_mintlify_syntax(content: str) -> str:
    """Remove Mintlify-specific syntax and convert to standard Markdown"""

    # Remove Mintlify frontmatter icon field
    content = re.sub(r"^icon:\s*['\"]?[\w-]+['\"]?\s*\n", "", content, flags=re.MULTILINE)

    # Convert CodeGroup to simple code blocks
    content = re.sub(r'<CodeGroup>\s*', '', content)
    content = re.sub(r'</CodeGroup>\s*', '', content)

    # Convert Tabs to sections
    content = re.sub(r'<Tabs>\s*', '', content)
    content = re.sub(r'</Tabs>\s*', '', content)
    content = re.sub(r'<Tab title="([^"]+)">\s*', r'**\1:**\n\n', content)
    content = re.sub(r'</Tab>\s*', '\n', content)

    # Convert Warning to admonition
    content = re.sub(r'<Warning>\s*', ':::warning\n\n', content)
    content = re.sub(r'</Warning>\s*', '\n:::\n\n', content)

    # Convert Note to admonition
    content = re.sub(r'<Note>\s*', ':::note\n\n', content)
    content = re.sub(r'</Note>\s*', '\n:::\n\n', content)

    # Convert Tip to admonition
    content = re.sub(r'<Tip>\s*', ':::tip\n\n', content)
    content = re.sub(r'</Tip>\s*', '\n:::\n\n', content)

    # Convert Info to admonition
    content = re.sub(r'<Info>\s*', ':::info\n\n', content)
    content = re.sub(r'</Info>\s*', '\n:::\n\n', content)

    # Remove ParamField - convert to table format later
    content = re.sub(r'<ParamField[^>]*>\s*', '', content)
    content = re.sub(r'</ParamField>\s*', '\n', content)
    content = re.sub(r'<Expandable[^>]*>\s*', '', content)
    content = re.sub(r'</Expandable>\s*', '', content)

    # Remove ResponseField
    content = re.sub(r'<ResponseField[^>]*>\s*', '', content)
    content = re.sub(r'</ResponseField>\s*', '\n', content)

    # Convert AccordionGroup to sections
    content = re.sub(r'<AccordionGroup>\s*', '', content)
    content = re.sub(r'</AccordionGroup>\s*', '', content)
    content = re.sub(r'<Accordion title="([^"]+)">\s*', r'### \1\n\n', content)
    content = re.sub(r'</Accordion>\s*', '\n', content)

    # Convert CardGroup to list
    content = re.sub(r'<CardGroup[^>]*>\s*', '', content)
    content = re.sub(r'</CardGroup>\s*', '', content)

    # Convert Card to link
    def card_to_link(match):
        title = match.group(1)
        href = match.group(2) if match.group(2) else ''
        return f'- [{title}]({href})'

    content = re.sub(
        r'<Card\s+title="([^"]+)"[^>]*href="([^"]*)"[^>]*>\s*[^<]*</Card>',
        card_to_link,
        content
    )
    content = re.sub(r'<Card[^>]*>\s*', '', content)
    content = re.sub(r'</Card>\s*', '\n', content)

    # Fix API group references
    content = content.replace('infra.operator.aws.io', 'aws-infra-operator.runner.codes')

    # Clean up multiple newlines
    content = re.sub(r'\n{4,}', '\n\n\n', content)

    # Convert .mdx extension references to .md
    content = content.replace('.mdx)', '.md)')
    content = content.replace('.mdx]', '.md]')

    return content


def update_frontmatter(content: str, sidebar_position: int = None) -> str:
    """Update frontmatter for Docusaurus"""

    # Check if has frontmatter
    if not content.startswith('---'):
        return content

    # Find end of frontmatter
    end = content.find('---', 3)
    if end == -1:
        return content

    frontmatter = content[3:end].strip()
    body = content[end+3:].strip()

    # Parse frontmatter
    lines = frontmatter.split('\n')
    new_lines = []
    has_sidebar = False

    for line in lines:
        # Skip icon field
        if line.strip().startswith('icon:'):
            continue
        # Keep title and description
        if line.strip().startswith('title:') or line.strip().startswith('description:'):
            new_lines.append(line)
        if line.strip().startswith('sidebar_position:'):
            has_sidebar = True
            new_lines.append(line)

    # Add sidebar position if not present
    if not has_sidebar and sidebar_position is not None:
        new_lines.append(f'sidebar_position: {sidebar_position}')

    new_frontmatter = '\n'.join(new_lines)

    return f'---\n{new_frontmatter}\n---\n\n{body}'


def migrate_file(src: Path, dst: Path, sidebar_position: int = None):
    """Migrate a single file"""
    print(f"Migrating: {src.name} -> {dst}")

    content = src.read_text()
    content = clean_mintlify_syntax(content)
    content = update_frontmatter(content, sidebar_position)

    dst.parent.mkdir(parents=True, exist_ok=True)
    dst.write_text(content)


def migrate_directory(src_dir: Path, dst_dir: Path):
    """Migrate a directory of files"""

    if not src_dir.exists():
        return

    for i, item in enumerate(sorted(src_dir.iterdir()), start=1):
        if item.name in SKIP:
            continue

        if item.is_file() and item.suffix in ['.md', '.mdx']:
            # Determine target name
            dst_name = item.stem + '.md'
            dst_path = dst_dir / dst_name
            migrate_file(item, dst_path, sidebar_position=i)
        elif item.is_dir():
            migrate_directory(item, dst_dir / item.name)


def main():
    print("Starting Mintlify to Docusaurus migration...")

    # Migrate individual files
    for src_name, dst_name in PATH_MAPPING.items():
        src = SOURCE_DIR / src_name
        dst = TARGET_DIR / dst_name
        if src.exists():
            migrate_file(src, dst)

    # Migrate services directory
    services_src = SOURCE_DIR / 'services'
    services_dst = TARGET_DIR / 'services'
    if services_src.exists():
        migrate_directory(services_src, services_dst)

    # Migrate features directory
    features_src = SOURCE_DIR / 'features'
    features_dst = TARGET_DIR / 'features'
    if features_src.exists():
        migrate_directory(features_src, features_dst)

    # Migrate guides directory
    guides_src = SOURCE_DIR / 'guides'
    guides_dst = TARGET_DIR / 'guides'
    if guides_src.exists():
        migrate_directory(guides_src, guides_dst)

    print("Migration complete!")


if __name__ == '__main__':
    main()
