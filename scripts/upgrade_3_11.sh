#!/usr/bin/env bash

# 登录cloudpods控制节点获取mysql账号密码信息,填写到下面变量
# kubectl get oc -n onecloud default -o yaml | grep -A 4 mysql
#
# 升级到3.11版本之后,等待所有华为云账号同步完成后,执行此升级脚本
# 脚本原理如下:
# 脚本会检索1天内deleted=1的华为云资源，通过external_id反查当前数据库中对应external_id相同且deleted=0的资源
# 先会将deleted=1的资源id设为id=id-old, 再将deleted=0的资源id变更到deleted=1的资源id
#
HOST='127.0.0.1'
USERNAME='root'
PASSWORD=''
DATETIME=''
if ["$(uname)"=="Darwin"]; then
    DATETIME=$(date -v-1d -u "+%Y-%m-%dT%H:%M%SZ")
else
    DATETIME=$(date -d "1 day ago" -u "+%Y-%m-%dT%H:%M%SZ")
if 

function exec_sql_with_db() {
    export MYSQL_PWD=$PASSWORD
    local db=$1
    local sql=$2
    echo $(mysql -u"$USERNAME" -h "$HOST" -D"$db" -s -e "$sql")
}

function get_relation_tables() {
    local field=$1
    exec_sql_with_db "information_schema" "select table_name from columns where column_name='$field' and table_schema='yunioncloud'"
}

function exec_sql() {
    exec_sql_with_db "yunioncloud" "$1"
}

function uuid() {
    echo $1 | awk -F '-' '{print $1"-"$2"-"$3"-"$4"-"$5}'
}

function change_uuid() {
    local table=$1
    local target_id=$2

    uid=$(uuid $target_id) 
    if [ "$uid" != "$target_id" ]; then
        target_id=$uid
    else
        exec_sql "update $table set id='$target_id-old' where id='$target_id'"
    fi
    echo $target_id
}


echo "upgrade guests_tbl"
exec_sql "select name, id, external_id from guests_tbl where deleted=1 and length(external_id) > 0 and host_id in (select id from hosts_tbl where deleted=1 and manager_id in (select id from cloudproviders_tbl where deleted=1 and provider='Huawei' and deleted_at > '$DATETIME'))" | while read -r line; do
    info=($(echo $line | tr " ", "\n"))
    name=${info[0]}
    target_id=${info[1]}
    external_id=${info[2]}
    id=$(exec_sql "select id from guests_tbl where deleted=0 and external_id='$external_id' and host_id in (select id from hosts_tbl where deleted=0 and manager_id in (select id from cloudproviders_tbl where deleted=0 and provider='Huawei'))")
    if [ -n "$id" ]; then
        target_id=$(change_uuid "guests_tbl" $target_id)
        echo "change server $name id from $id => $target_id"

        exec_sql "update guests_tbl set id='$target_id' where id='$id' and deleted=0"

        tables=$(get_relation_tables "guest_id")
        for table in $tables; do
            exec_sql "update $table set guest_id='$target_id' where guest_id='$id' and deleted=0"
        done

    fi

done

echo "upgrade disks_tbl"
exec_sql "select name, id, external_id from disks_tbl where deleted=1 and length(external_id) > 0 and storage_id in (select id from storages_tbl where deleted=1 and manager_id in (select id from cloudproviders_tbl where deleted=1 and provider='Huawei' and deleted_at > '$DATETIME'))" | while read -r line; do
    info=($(echo $line | tr " ", "\n"))
    name=${info[0]}
    target_id=${info[1]}
    external_id=${info[2]}
    id=$(exec_sql "select id from disks_tbl where deleted=0 and external_id='$external_id' and storage_id in (select id from storages_tbl where deleted=0 and manager_id in (select id from cloudproviders_tbl where deleted=0 and provider='Huawei'))")
    if [ -n "$id" ]; then
        target_id=$(change_uuid "disks_tbl" $target_id)
        echo "change disk $name id from $id => $target_id"

        exec_sql "update disks_tbl set id='$target_id' where id='$id' and deleted=0"

        tables=$(get_relation_tables "disk_id")
        for table in $tables; do
            exec_sql "update $table set disk_id='$target_id' where disk_id='$id' and deleted=0"
        done

    fi

done



echo "upgrade networks_tbl"
exec_sql "select name, id, external_id from networks_tbl where deleted=1 and length(external_id) > 0 and wire_id in (select id from wires_tbl where deleted=1 and vpc_id in (select id from vpcs_tbl where deleted=1 and manager_id in (select id from cloudproviders_tbl where deleted=1 and provider='Huawei' and deleted_at > '$DATETIME')))" | while read -r line; do
    info=($(echo $line | tr " ", "\n"))
    name=${info[0]}
    target_id=${info[1]}
    external_id=${info[2]}
    id=$(exec_sql "select id from networks_tbl where deleted=0 and external_id='$external_id' and wire_id in (select id from wires_tbl where deleted=0 and vpc_id in (select id from vpcs_tbl where deleted=0 and manager_id in (select id from cloudproviders_tbl where deleted=0 and provider='Huawei')))")
    if [ -n "$id" ]; then
        target_id=$(change_uuid "networks_tbl" $target_id)
        echo "change network $name id from $id => $target_id"

        exec_sql "update networks_tbl set id='$target_id' where id='$id' and deleted=0"

        tables=$(get_relation_tables "network_id")
        for table in $tables; do
            if [ "$table" == "network_additional_wire_tbl" ]; then
                exec_sql "update $table set network_id='$target_id' where network_id='$id'"
            else
                exec_sql "update $table set network_id='$target_id' where network_id='$id' and deleted=0"
            fi
        done

    fi
done


function upgrade_managed_resources_table() {
    local res_type=$1
    local table_name=$2
    local field=$3

    echo "upgrade $table_name"
    exec_sql "select name, id, external_id from $table_name where deleted=1 and length(external_id) > 0 and manager_id in (select id from cloudproviders_tbl where deleted=1 and provider='Huawei' and deleted_at > '$DATETIME')" | while read -r line; do
        info=($(echo $line | tr " ", "\n"))
        name=${info[0]}
        target_id=${info[1]}
        external_id=${info[2]}
        id=$(exec_sql "select id from $table_name where deleted=0 and external_id='$external_id' and manager_id in (select id from cloudproviders_tbl where deleted=0 and provider='Huawei')")
        if [ -n "$id" ]; then
            target_id=$(change_uuid $table_name $target_id)
            echo "change $res_type $name id from $id => $target_id"
    
            exec_sql "update $table_name set id='$target_id' where id='$id' and deleted=0"
    
            tables=$(get_relation_tables $field)
            for table in $tables; do
                exec_sql "update $table set $field='$target_id' where $field='$id' and deleted=0"
            done
        fi
    done
}


upgrade_managed_resources_table "elasticcacheinstance" "elasticcacheinstances_tbl" "elasticcache_id"
upgrade_managed_resources_table "dbinstance" "dbinstances_tbl" "dbinstance_id"
upgrade_managed_resources_table "bucket" "buckets_tbl" "bucket_id"
upgrade_managed_resources_table "vpc" "vpcs_tbl" "vpc_id"
upgrade_managed_resources_table "cdn" "cdn_domains_tbl" "cdn_domain_id"
upgrade_managed_resources_table "dns" "dnszones_tbl" "dns_zone_id"
upgrade_managed_resources_table "elastic_search" "elastic_searchs_tbl" "elastic_search_id"
upgrade_managed_resources_table "eip" "elasticips_tbl" "eip_id"
upgrade_managed_resources_table "inter_vpc_network" "inter_vpc_networks_tbl" "inter_vpc_network_id"
upgrade_managed_resources_table "kafka" "kafkas_tbl" "kafka_id"
upgrade_managed_resources_table "loadbalancer" "loadbalancers_tbl" "loadbalancer_id"
upgrade_managed_resources_table "snapshot" "snapshots_tbl" "snapshot_id"
upgrade_managed_resources_table "sslcertificate" "sslcertificates_tbl" "sslcertificate_id"


