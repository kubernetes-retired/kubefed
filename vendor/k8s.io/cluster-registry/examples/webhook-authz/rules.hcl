# kube-system default user can access any namespace
access "allow" {
    username = "system:serviceaccount:kube-system:default"
    verb = "(list|watch|get)"
}

access "allow" {
    username = "admin"
}

access "allow" {
    username = "testuser"
    verb = "(list|watch|get)"
}
