import socket


def test_endpoint(host, port=80, timeout=2):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    try:
        s.settimeout(timeout)
        s.connect((host, port))
        print('test for %s true' % host)
        return True
    except socket.error:
        print('test for %s false' % host)
        return False
    finally:
        s.close()


test_endpoint('juicefs-testaliyun.oss-cn-shenzhen-internal.aliyuncs.com')
