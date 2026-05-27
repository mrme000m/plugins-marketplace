rule Phishing_Indicators {
    meta:
        description = "Detects common phishing keywords in documents"
        author = "mrme000m"
    strings:
        $s1 = "account suspended" nocase
        $s2 = "immediate action required" nocase
        $s3 = "verify your identity" nocase
        $s4 = "click here to log in" nocase
        $s5 = "unauthorized access detected" nocase
        $s6 = "security alert" nocase
    condition:
        2 of them
}

rule Suspicious_Office_OLE {
    meta:
        description = "Detects suspicious OLE objects/macros"
    strings:
        $v1 = "VBA"
        $v2 = "AutoOpen"
        $v3 = "Workbook_Open"
        $v4 = "Shell"
        $v5 = "WScript.Shell"
    condition:
        uint32(0) == 0xE11AB1D0 and any of them
}
