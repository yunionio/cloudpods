export PATH=/opt/yunion/bin:$PATH

if [ -f /etc/profile.d/bash_completion.sh ]; then
    source /etc/profile.d/bash_completion.sh
fi

test -f /etc/yunion/rcadmin && source /etc/yunion/rcadmin

if [ -n "$OS_AUTH_TOKEN" ]; then
    test -n "$OS_USERNAME" && unset OS_USERNAME
    test -n "$OS_PASSWORD" && unset OS_PASSWORD
    test -n "$OS_DOMAIN_NAME" && unset OS_DOMAIN_NAME
    test -n "$OS_ACCESS_KEY" && unset OS_ACCESS_KEY
    test -n "$OS_SECRET_KEY" && unset OS_SECRET_KEY
fi

source <(climc --completion bash)
source <(kubectl completion bash)

echo "Welcome to Cloud Shell :-) You may execute climc and other command tools in this shell."
echo "Please exec 'climc' to get started"
echo ""
