#!/bin/bash

set -e

# Setup base system
pacman -Syu --noconfirm openssh git gettext binutils
sed -i "s/INTEGRITY_CHECK=.*$/INTEGRITY_CHECK=(sha256)/" /etc/makepkg.conf
useradd -ms /bin/bash aur
su -m aur <<'EOSU'

set -e

# Configuration
export HOME=/home/aur
export REPO_URL="ssh://aur@aur.archlinux.org/$PACKAGE_NAME.git"
export NEW_RELEASE="${COMMIT_REF##*/v}"
export COMMIT_MESSAGE="$(echo $COMMIT_MESSAGE | envsubst)"
echo "---------------- AUR Package version $PACKAGE_NAME/$NEW_RELEASE ----------------"

# SSH & GIT Setup
mkdir "$HOME/.ssh" && chmod 700 "$HOME/.ssh"
ssh-keyscan -t ed25519 aur.archlinux.org >> "$HOME/.ssh/known_hosts"
echo -e "$SSH_PRIVATE_KEY" | base64 -d > "$HOME/.ssh/id_rsa"
chmod 600 "$HOME/.ssh/id_rsa"
git config --global user.name "$COMMIT_USERNAME"
git config --global user.email "$COMMIT_EMAIL"

# Clone AUR Package
cd /tmp
echo "$REPO_URL"
git clone "$REPO_URL"
cd "$PACKAGE_NAME"

# Generate a dummy PKGBUILD so we can grab the latest releases SHA256SUMS
cat PKGBUILD.template | envsubst '$NEW_RELEASE' > PKGBUILD

export SHA256_SUMS_x86_64="$(CARCH=x86_64 makepkg -g 2> /dev/null)"
echo "SHA256_SUMS_x86_64: $SHA256_SUMS_x86_64"

export SHA256_SUMS_aarch64="$(CARCH=aarch64 makepkg -g 2> /dev/null)"
echo "SHA256_SUMS_aarch64: $SHA256_SUMS_aarch64"

export SHA256_SUMS_arm="$(CARCH=arm makepkg -g 2> /dev/null)"
echo "SHA256_SUMS_arm: $SHA256_SUMS_arm"

cat PKGBUILD.template | envsubst '$NEW_RELEASE$SHA256_SUMS_x86_64$SHA256_SUMS_aarch64$SHA256_SUMS_arm' > PKGBUILD
makepkg --printsrcinfo > .SRCINFO

echo "------------- BUILD DONE ----------------"
git add PKGBUILD .SRCINFO
git commit -m "$COMMIT_MESSAGE"
git push
echo "------------- PUBLISH DONE ----------------"
EOSU
