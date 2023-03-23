import os
import argparse
from xdis.unmarshal import load_code
from xdis.load import write_bytecode_file
from xdis.codetype import Code3


def get_codeobject_list(pycfile, magic):
    co_list = []

    def recurse_co(co):
        co_list.append(co)
        for const in co.co_consts:
            if isinstance(const, Code3):
                recurse_co(const)

    f = open(pycfile, "rb")
    f.seek(16, os.SEEK_SET)
    codeobject = f.read()
    f.close()
    root_co = load_code(codeobject, magic)
    recurse_co(root_co)

    return co_list


def main():
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="action")

    list_parser = subparsers.add_parser("list", help="List all codeobjects in the pyc file")
    list_parser.add_argument("--magic", required=True, help="The python magic", type=int)
    list_parser.add_argument("pycfile", help="The path to the .pyc file")

    build_parser = subparsers.add_parser("build", help="Create a code object")
    build_parser.add_argument("--magic", required=True, help="The python magic", type=int)
    build_parser.add_argument("--index", required=True, help="The zero based index of the code object", type=int)
    build_parser.add_argument("pycfile", help="The path to the .pyc file")
    build_parser.add_argument("outfile", help="The output file path")
    build_parser.add_argument("co_bytes", help="The code object bytes to assemble as a hex string")

    args = parser.parse_args()

    if args.action == "list":
        co_list = get_codeobject_list(args.pycfile, args.magic)
        for co in co_list:
            print(f"{co.co_name}:{len(co.co_code)}")

        
    elif args.action == "build":
        co_list = get_codeobject_list(args.pycfile, args.magic)
        code_object = co_list[args.index]
        new_code_bytes = bytes.fromhex(args.co_bytes)
        code_object.co_code = new_code_bytes
        write_bytecode_file(args.outfile, code_object, args.magic)


if __name__ == "__main__":
    main()

