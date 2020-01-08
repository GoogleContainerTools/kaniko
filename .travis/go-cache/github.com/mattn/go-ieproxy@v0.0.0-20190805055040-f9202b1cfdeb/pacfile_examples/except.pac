function FindProxyForURL(url, host) {
    var useSocks = ["imgur.com"];

    for (var i= 0; i < useSocks.length; i++) {
        if (shExpMatch(host, useSocks[i])) {
            return "PROXY localhost:9999";
        }
    }

    return "DIRECT";
}
