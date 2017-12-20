struct PingReply {
    1: string value
    2: string err
}

service HackerService {
    PingReply Ping()
}
