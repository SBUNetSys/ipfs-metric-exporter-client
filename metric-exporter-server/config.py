import json

if __name__ == '__main__':
    new_config = '{"Plugins":{"Plugins":{"metric-export-plugin":{"Config":{"PopulatePrometheusInterval":10,"AgentVersionCutOff":20,"TCPServerConfig":{"ListenAddresses":["0.0.0.0:4321"]},"HTTPServerConfig":{"ListenAddresses":["0.0.0.0:4322"]}}}}}}'
    new_config_json = json.loads(new_config)
    with open(".ipfs/config", 'r') as fin:
        data = json.load(fin)
        data["Plugins"] = new_config_json["Plugins"]
        # "Gateway": "/ip4/0.0.0.0/tcp/8080",
        data["Addresses"]["Gateway"] = "/ip4/0.0.0.0/tcp/8080"
        data["Addresses"]["API"] = "/ip4/0.0.0.0/tcp/5001"
    with open(".ipfs/config", 'w') as fout:
        json.dump(data, fout, indent=4)
