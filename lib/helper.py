import os
import argparse

def get_codeobject_list(pycfile, magic, backend):
    if backend == "xdis":
        import sys
        sys.path.append("xdis.zip")

        from xdis.unmarshal import load_code
        from xdis.codetype import Code3
    
    elif backend == "native":
        from marshal import loads
        from types import CodeType as Code3

        def load_code(data, _):
            return loads(data)


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


def build_pyc(pycfile, magic, backend, index, co_bytes, outfile):
    if backend == "xdis":
        import sys
        sys.path.append("xdis.zip")

        from xdis.load import write_bytecode_file
    elif backend == "native":
        from marshal import dump

        def write_bytecode_file(outfile, code_object, magic):
            f = open(outfile, "wb")
            f.write(magic.to_bytes(length=2, byteorder="little") + b"\r\n")
            f.write(b"\0" * 12)
            dump(code_object, f)
            f.close()


    co_list = get_codeobject_list(pycfile, magic, backend)
    code_object = co_list[index]
    new_code_bytes = bytes.fromhex(co_bytes)

    if backend == "xdis":
        code_object.co_code = new_code_bytes
    elif backend == "native":
        code_object = code_object.replace(co_code=new_code_bytes)

    write_bytecode_file(outfile, code_object, magic)

def main():
    parser = argparse.ArgumentParser()
    subparsers = parser.add_subparsers(dest="action")

    list_parser = subparsers.add_parser("list", help="List all codeobjects in the pyc file")
    list_parser.add_argument("--magic", required=True, help="The python magic", type=int)
    list_parser.add_argument("--backend", required=True, choices=["xdis", "native"])
    list_parser.add_argument("pycfile", help="The path to the .pyc file")

    build_parser = subparsers.add_parser("build", help="Create a code object")
    build_parser.add_argument("--magic", required=True, help="The python magic", type=int)
    build_parser.add_argument("--backend", required=True, choices=["xdis", "native"])
    build_parser.add_argument("--index", required=True, help="The zero based index of the code object", type=int)
    build_parser.add_argument("pycfile", help="The path to the .pyc file")
    build_parser.add_argument("outfile", help="The output file path")
    build_parser.add_argument("co_bytes", help="The code object bytes to assemble as a hex string")

    args = parser.parse_args()

    if args.action == "list":
        co_list = get_codeobject_list(args.pycfile, args.magic, args.backend)
        for co in co_list:
            print(f"{co.co_name}:{len(co.co_code)}")

    elif args.action == "build":
        build_pyc(args.pycfile, args.magic, args.backend, args.index, args.co_bytes, args.outfile)


if __name__ == "__main__":
    main()
