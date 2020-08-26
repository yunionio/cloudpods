#!/bin/bash

add_host_specific_libraries() {
    for libraries in 'libsoftokn3.so' 'libsqlite3.so.0' 'libfreeblpriv3.so'; do
        lib_path=""
        for base_lib_path in '/lib64/' '/lib/' '/usr/lib/x86_64-linux-gnu/' '/lib/x86_64-linux-gnu/'; do
            if [ ! -e $base_lib_path ]; then
                continue
            fi
            lib_path="$(find $base_lib_path -name $libraries -print -quit)"
            if [ -z "$lib_path" ]; then
                continue
            fi
            real_lib_path="$(readlink -f $lib_path)"
            cp $real_lib_path $1/lib/$libraries
            break
        done
        if [ -z "$lib_path" ]; then
            echo "failed find $libraries ..."
            exit 1
        fi
    done
}

add_host_specific_libraries $@