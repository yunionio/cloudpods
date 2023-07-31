export PATH=/opt/yunion/bin:$PATH

test -f /etc/yunion/rcadmin && source /etc/yunion/rcadmin

source <(climc --completion bash)

echo "Welcome to Cloud Shell :-) You may execute climc and other command tools in this shell."
echo "Please exec 'climc' to get started"
echo ""
