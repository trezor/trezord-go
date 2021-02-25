import requests


def api_call(endpoint, data=None):
    API = "http://127.0.0.1:21325"
    headers = {"origin": "https://test.trezor.io"}

    r = requests.post(API + endpoint, headers=headers, data=data)
    if data:
        return r.text
    else:
        return r.json()


j = api_call("/")
print(j)


while True:
    j = api_call("/enumerate")
    print(j)

    paths = [x["path"] for x in j]
    for path in paths:
        j = api_call(f"/acquire/{path}/null")
        print(j)
        sess = j["session"]

        d = api_call(f"/call/{sess}", "000000000000")  # send Initialize
        print(d)

        j = api_call(f"/release/{sess}")
        print(j)
