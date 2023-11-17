export PATH=/opt/yunion/bin:$PATH

if [ -f /etc/profile.d/bash_completion.sh ]; then
    source /etc/profile.d/bash_completion.sh
fi

test -f /etc/yunion/rcadmin && source /etc/yunion/rcadmin

source <(climc --completion bash)
source <(kubectl completion bash)

echo "Welcome to Cloud Shell :-) You may execute climc and other command tools in this shell."
echo "Please exec 'climc' to get started"
echo ""
