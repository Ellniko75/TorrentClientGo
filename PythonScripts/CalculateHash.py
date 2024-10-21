import bencoding
import hashlib
import sys

with open(f"torrents/{sys.argv[1]}", "rb") as f:
    data = bencoding.bdecode(f.read())
    info = data[b'info']
    hashed_info = hashlib.sha1(bencoding.bencode(info)).hexdigest()
    print(hashed_info)

