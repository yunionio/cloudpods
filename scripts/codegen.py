#!/usr/bin/env python

import argparse
import os
from os.path import join as pjoin
import subprocess
import sys


def run_cmd(cmds):
    print(' '.join(cmds))
    proc = subprocess.Popen(
        cmds, stdout=subprocess.PIPE)
    while True:
        line_gen = proc.stdout.readline().rstrip()
        if not line_gen:
            break
        for l in line_gen:
            print(l)


def run_generator(tool, input_dirs, out_pkg):
    cmd = [tool]
    for i in input_dirs:
        cmd.extend(["--input-dirs", i])
    cmd.extend(["--output-package", out_pkg])
    run_cmd(cmd)


def run_model_api(input_dirs, out_pkg):
    return run_generator('model-api-gen', input_dirs, out_pkg)


def run_swagger_gen(input_dirs, out_pkg):
    return run_generator('swagger-gen', input_dirs, out_pkg)


def run_swagger_serve(output_dir):
    def is_swagger_yaml(file):
        return file.endswith('.yaml') and file.startswith('swagger')
    yamls = [pjoin(output_dir, x) for x in os.listdir(output_dir) if is_swagger_yaml(x)]
    cmd = ['swagger-serve', 'generate']
    cmd.extend(['-i', ','.join(yamls), '-o', output_dir])
    cmd.append('--serve')
    run_cmd(cmd)


def run_swagger_yaml(svc, swagger_pkg_dir, output_dir):
    if not os.path.exists(output_dir):
        os.makedirs(output_dir)
    work_dir = pjoin(swagger_pkg_dir, svc)
    cmd = ["swagger", "generate", "spec", "--scan-models"]
    cmd.extend(["--work-dir", work_dir])
    cmd.extend(["-o", pjoin(output_dir, "swagger_%s.yaml" % svc)])
    run_cmd(cmd)


class FuncDispatcher(object):

    def __init__(self):
        self.gen_dict = {}
        self.collect_funcs()

    def collect_funcs(self):
        gen_dict = {}
        for attr in dir(self):
            if not attr.startswith('gen_'):
                continue
            func = getattr(self, attr)
            if not callable(func):
                continue
            svc = attr.lstrip('gen_')
            gen_dict[svc] = func
        self.gen_dict = gen_dict

    def get_choices(self):
        svc = ["all"]
        svc.extend(self.gen_dict.keys())
        return svc

    def dispatcher(self, opt):
        def run_one(key):
            self.gen_dict[key]()
        def run_all():
            for key in self.gen_dict:
                run_one(key)
        if opt.service is None or 'all' in opt.service:
            run_all()
        else:
            for svc in opt.service:
                run_one(svc)

    def get_parser(self, subparsers, name, help):
        parser = subparsers.add_parser(name, help=help)
        parser.add_argument("-s", "--service", help="%s for services" % help,
                            nargs='+', choices=self.get_choices())
        parser.set_defaults(func=self.dispatcher)


class ModelAPI(FuncDispatcher):

    def __init__(self, pkg_dir, apis_dir):
        super(ModelAPI, self).__init__()
        self.pkg_dir = pkg_dir
        self.apis_dir = apis_dir

    def get_parser(self, subparsers):
        return super(ModelAPI, self).get_parser(
            subparsers, "model-api", "generate model struct code for api")

    def run(self, pkg=[], out=[]):
        if pkg is None:
            return
        in_dir = pjoin(self.pkg_dir, *pkg)
        out_dir = pjoin(self.apis_dir, *out)
        run_model_api([in_dir], out_dir)

    def run_same(self, pkg):
        self.run(pkg=[pkg], out=[pkg])

    def run_model(self, svc):
        self.run(pkg=[svc, "models"], out=[svc])

    def gen_cloudcommon(self):
        self.run(pkg=["cloudcommon", "db"])

    def gen_cloudprovider(self):
        self.run_same("cloudprovider")

    def gen_compute(self):
        self.run_model("compute")

    def gen_image(self):
        self.run_model("image")

    def gen_identity(self):
        self.run(pkg=["keystone", "models"], out=["identity"])


class SwaggerCode(FuncDispatcher):

    def __init__(self, pkg_dir, pkg_swagger):
        super(SwaggerCode, self).__init__()
        self.pkg_dir = pkg_dir
        self.pkg_swagger = pkg_swagger

    def get_parser(self, subparsers):
        return super(SwaggerCode, self).get_parser(
                subparsers, "swagger-code", "generate swagger code")

    def run(self, svc, pkg=[], out=''):
        if svc is None:
            return
        if pkg is None:
            return
        svc_pkg_dir = pjoin(self.pkg_dir, svc)
        input_dirs = [pjoin(svc_pkg_dir, x) for x in pkg]
        out_dir = pjoin(self.pkg_swagger, out)
        run_swagger_gen(input_dirs, out_dir)

    def run_svc(self, svc, pkg=[]):
        self.run(svc, pkg=pkg, out=svc)

    def gen_identity(self):
        self.run("keystone", pkg=["tokens", "models"], out="identity")

    def gen_compute(self):
        self.run_svc("compute", pkg=["models"])

    def gen_image(self):
        self.run_svc("image", pkg=["models"])


class SwaggerYAML(FuncDispatcher):

    def __init__(self, swagger_dir, out_dir):
        super(SwaggerYAML, self).__init__()
        self.swagger_dir = swagger_dir
        self.out_dir = out_dir

    def get_parser(self, subparsers):
        return super(SwaggerYAML, self).get_parser(
                subparsers, "swagger-yaml", "generate swagger yaml")

    def run(self, svc):
        run_swagger_yaml(svc, self.swagger_dir, self.out_dir)

    def gen_identity(self):
        self.run("identity")

    def gen_compute(self):
        self.run("compute")

    def gen_image(self):
        self.run("image")


class SwaggerServe(object):

    def __init__(self, output_dir):
        self.output_dir = output_dir

    def get_parser(self, subparsers):
        parser = subparsers.add_parser(
                "swagger-serve", help="generate swagger web site")
        parser.set_defaults(func=self.run)

    def run(self, opt):
        run_swagger_serve(self.output_dir)


if __name__ == "__main__":
    ONECLOUD="yunion.io/x/onecloud"
    PKG_ONECLOUD=pjoin(ONECLOUD, "pkg")
    PKG_APIS=pjoin(PKG_ONECLOUD, "apis")
    PKG_GENERATED=pjoin(PKG_ONECLOUD, "generated")
    PKG_SWAGGER=pjoin(PKG_GENERATED, "swagger")
    SCRIPTS_DIR=os.path.dirname(os.path.realpath(__file__))
    CUR_DIR=os.path.dirname(SCRIPTS_DIR)
    SWAGGER_PKG=pjoin(CUR_DIR, "pkg", "generated", "swagger")
    OUTPUT_DIR=pjoin(CUR_DIR, "_output")
    OUTPUT_SWAGGER_DIR=pjoin(OUTPUT_DIR, "swagger")

    model_api = ModelAPI(PKG_ONECLOUD, PKG_APIS)
    swagger_yaml = SwaggerYAML(SWAGGER_PKG, OUTPUT_SWAGGER_DIR)
    swagger_code = SwaggerCode(PKG_ONECLOUD, PKG_SWAGGER)
    swagger_serve = SwaggerServe(OUTPUT_SWAGGER_DIR)

    parser = argparse.ArgumentParser(description="Code generate helper.")
    subparsers = parser.add_subparsers(dest='cmd')
    subparsers.required = True

    for cmd in [model_api, swagger_code, swagger_yaml, swagger_serve]:
        cmd.get_parser(subparsers)

    if len(sys.argv) == 1:
        parser.print_help()
        sys.exit(1)

    options = parser.parse_args()
    options.func(options)
