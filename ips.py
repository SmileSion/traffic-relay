start_ip = 120
end_ip = 138
base_ip_prefix = "172.18.50."
port = 8080

backend_urls = [
    f"http://{base_ip_prefix}{i}:{port}" for i in range(start_ip, end_ip + 1)
]

print("backend_urls =", backend_urls)
