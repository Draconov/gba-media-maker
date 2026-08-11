(function(global){
const FRAME_WIDTH = 120;
const FRAME_HEIGHT = 80;
const FRAME_BYTES = FRAME_WIDTH * FRAME_HEIGHT;
const RGB24_FRAME_BYTES = FRAME_BYTES * 3;
const { glyphBits } = global.GBAText;
const OCEAN_PALETTE_B64 = "KxULEQsNChEKFQkZ6xDrDOoU6RjqEMoMCB0HIecg5iDlJMcYxhjFIMUcwyDDIKMcpRijHKMcohyiHIIcohyBHIEcYRhhGGEYYRhgGGAYYBhAGEAUQBQgDCAIIAjHGMYYxSDFHMMgwyCjHKUYoxyjHKIcohyCHKIcgRyBHGEYYRhhGGEYYBhgGGAYQBhAFEAUIAwgCCAIxxjGGMUgxRzDIMMgoxylGKMcoxyiHKIcghyiHIEcgRxhGGEYYRhhGGAYYBhgGEAYQBRAFCAMIAggCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMYY/39/Ax8A4AM=";
const OCEAN_PIXELS_B64 = "BgYGAgYGCQ4KCgkQDgwQEBAQDhAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQAgICAgICAgIGBgYICAYGDxAQBgYGCggOCAgIEBAQEBAQEBAQEA8QEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQAAAAAAAAAAABAAEGBgYGBgoIBQgJBgYGBgYGBgYICAYGBggOEA4MCQgFEBAPEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQDxAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQDgAAAAAAAAAAAAAAAAEAAgICAgYGCgoGBgECAgYKCgoKCgYCBgoQEA8PBgYIBQ8GBhAQCAgQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQDgMQAwgICBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQDg4ODg4AAAAAAAAAAAAAAAAAAAAAAQICAgICAgYCBgYGBgYGBgYGCgkFDwkKCgoICgYGCAoKDw8MDw8PDw8PDw8QEBADAwMQEBAQEBAQEBAQEBAQEAMGAwMJCAgQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQDg0NDQ0NDQ0NDQ0MAAAAAAAAAAAAAAAAAAAAAAAAAAIAAQIBBgYGBgYGBgkODAkKCgYGBgYKCgoKCA8PDgMJDw8PDxAQEAMDAxAIEBAQEBAQEBAQEBAQDAkJAwMDAwMQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQDg4NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQICBgoKBgIGBgICBgYGBgYGBgYGCgkOCAgFEA8PDw8OAwgJBQ8QEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBADAxAOEBAQEBAQEBAQEBAQDg0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQAAAAAADQwAAAAAAAAAAAAAAAAAAAAAAAAAAgIBBgICAQIGBgYGBgYGCgoGBgYGCggPDw8PDw8ODAgIDA8PDw8PDw8PDw8PDw8PDw8PEBAQEBAQEBAIDAMDEAgGEBAQEBAQDQ0NDQ0NDQ4NDQ0NDQ0NDQ0NDQ0NDQ0NDQAAAA0NDQAAAAAAAAUMAAAABQAAAAAAAAAAAAAAAAAAAAAAAgICAgYGAgYGBgYGCgoICAwPDw8ODAgICAgOBg8PDw8PDw8PDw8PDw8PDw8PDw8PDw8PEBAPDxAQEBAQDAwMDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQwMDAwNDQ0NBA0NBQwEAAAFBAAMDQAAAAAAAAAAAAAAAQEBAQECAgIGBggGCggICA4PDAkOCAkICggICQ8PDw4IDg8ODw8PDw8ODw8PDw8PDw8PDw8PDAwMDAwMDAwMBQUFBQwMDAwMDAwMDAwMDA0NDQ0NDQ0NDQ0NDQ0NDQ0NDQ0NDQwMDA0MDAwMDAwMDQwMAAAMDAwBBAAAAAAAAAEEAwMBDA4OBQMGBgwNBAYGCgYGBgoJDA4PDw8MCAgIDg8JCg4ODw8PDw8PDw8PAAEAAAUMDAwFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUNDQ0NDQ0NDQ0NDQ0NDAwMDAwMDAwMDAwMDAwMDAwMDAwMAQEBDA0NDAICBAIFCQ0OBQUODg8DBgYGBgYGBgYGBgIICgoKCgoKCggICAgICAkPDw8PBQUAAAAABAQEBAUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFDAwMDQ0NDQ0NDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAUBBQECAQIGCA0NDg4MBQUODggGBQ8ODg4MDgwOAwYGBgYGBgYGCggICAgIBQ8PBQUFBQUFBQAAAAAABQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMAwICAgEDBAwMCQMDBgYGBgYEBQ0ODgwGBAwMDAwGBgYGBgYGBgoICAgIBAQEBAQJCQkJCQkJAQEAAQEABQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMBAEBAQEBBQUGBgYGBAkGBgYDDQwMDAYGBgkNBQ0NDQMGBgYGBgYGBAQEBAQIBAQEBAQEBAQEAAEBAAEAAAkFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFCQkJCQkFDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMAQEBAwQEBgYGBgwMDAwMBgYGAwMIDAwMCQQJCQUJAwgICAgICAgEBAQEBAQEBAQEBAAAAAABAQEEBAQICAgDCAgEBQUFBQUFBQUFBQUFBQUFBQUFCQkJCQkJCQkJBAkJBQUMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAMJBgYBAwwMDAUDCAkJDQkGAw4ODg4OAQEBAgECAQEBAQEBAQEEBAQEBAQEBAgICAgICAgICAgICAgICAgICAMDCAMDCAQEBAQEBAQJCQkJCQkJCQkJBAQEBAQEBAQEBAQEBAUFCQUFDAwMDAwMDAwMBQUFBQUFBQUFBQUFBQUFBQUBAQMFCQkEAwMDAwgFAQECAQICAQEBAQEBAQEBAQEBAQgEBAgICAgICAgICAMICAgICAgICAMKChISEhISEhISEhIDAQEBAwQIBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBQUFDAUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQMDCAYFBQwMCgoKCgICAgIBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQICAgEDAwgIAwEBAQEBEhISEhISEhISEggBAQEBAQEBAQEDCAQEBAQEBAQEBAQEBAQEBAQEBAQEFhYJCQkFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFAgICAgIKCgoKCgoKAQEBAQEBAQEBAQEBAQEBAQICAgICCgoSEhISEhISEhISEhIKCgICAgICAgIKCgoKCgEBAQEBAQEBAQECAgEIBAQEBAQEBAQEHioqGw8JBBcEBAQEBAQECQUFBQUFBQUFBQUFBQUFBQUFBQUFAgICAgICAgICAgICAgICAgICAQEBAQEBAQICAgICAgECAgICAgICChISEhISEhISEhISCgoKAgICAgICAQEBAQEBAQEBAQEBAQICAgJPURsaHx8oKCkrLCskDgkFBAQUBAQEBAQEBAQEBAQFBQUFBQUFBQUFBQUFAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIBAQEBBFUfHx9YIiIfHx8lJiIpKisrKysrCAQEBAQEBAQEBAQEBAQEBAQEBQUFBQQEBAQEAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAB8lJSVYVVBVWCImJiYeHyIsJSUqKywqKisrKwQIBAQEBAQEBAQEBAQEBAQEBAQEBAQEBwcHBwcHBwcHBwcHBwcHBwcHBwcCAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICTgJPWB9RVVVVVVggH09PT1UeHhslKCwqJSIlKCgpKSorKSgICAAIBAQEBAQEBAQEBAQEBAQECwsHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcCAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgZPH1hRVVgfUE8QDg4OVVAQEBBNExUeHiIqLCwrKCUmIiIoKyosKSoqKwgIBAQEBAQEBAQEBAQEBwcLCwsLCwcHBwcHBwsHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHAgICAgICAgICAgICAgICAgICAgICAlVYUFBYWFAODk9VHyIlKygfIh9YTRMVFRUeJSkrKywoKygiIiYpKysoKioqKwEICAgEBAQEBAQEBwcHBwcHCwsLCwsLCwsLCwsLCwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHURBVUFVVDg5VJigrLS0tLS0tLS0tKSkpIh4VGxsgIicpKywqIiIiIigsKCoqKSoqKioqAwMICAgIBwcHBwcHBwcHBwcHBwcLCwsLCwsLCwsLCwsLCwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHBwcHB1MOEFVVWE8NUCgrLS0tLS0tLS0tLS0tLS0tLCoqKSkmJiIiIiUrKyolIiIiIiYqKyorLCsqKysoIwoKCwsLCwcHBwcHBwcHBwcHBwcHBwcLCwsLCwsLCwsLCwsLCwsLCwsLCwsLBwcHCwoKCgoSEkxMCgcHBwcHBw8gURAQUA0lKywsLCwtLS0tLS0tLS0tLS0tLS0tLS0tKystKigmKCkrKykoIiIiJiYmJikpKissKikqCwsLCwsLCwsLCwsHBwcHBwcHBwcHBwcHBwcHBwcLCwoKCgoKCgoSEhISCwsLCwsLCwsLCwsLCwsLCwtSTR9PDU9VDSkrLCwtLS0tLCwtLS0tLS0tLS0tLS0tLS0tLS0tLS0tLCwrKywsLCsqKSkpJSUlJSgqKSsrCwsLCwsLCwsLCwsLCwsLCwsLCwcHBwcHCwsLCwsHBwcHBwcHBwcHCwsKEhISEhISEgsKCwsLCwsLC1QFWE8QTVgQKSsrKioqKyosLSsrLSstLS0tLSorLS0tLS0tLS0tLS0tLS0tLS0tLS0tLS0tLC0sKyorKikqCwsLCwsLCwsLCwsLCwsLCwsLCwsLCwsLCwsLCwsLCwsLCwsLCwsLCwsHCwsLCwsLCgoKCkwKCwsLC1JVTQ8PEE0qKlILTCghISEoKykrKystKy0tKystLS0tLS0rLS0tLS0tLS0tLS0tLS0tLSwsKysrLCwsKy0sHhweIh8bHB4fIiYmKCceGxsfIB8fHx8kKCgoKSkoKCIbHh4eHxsbGxsgISEZGRoYCwsLCwsLCwsLCx9PDVUNTyoqCwsLC0whTFMhKSMpKysrLC0tKy0tLS0tKyssLS0tLS0tLS0tLS0tLS0tKysrKystKystLSsrHiUkGxUVHh8eHhYbHh4fFxsXGxsfJSkpKSkpKSgmKSkiHx4VFBUbHx8fHx4iJSUmKCcpJygoH1ZUEFUMUBAPKCtXVFdXIgZSCk0hJCQpKystLS0tLS0tLS0rLC0tLS0tLS0tLS0tLS0tLSsrKysrLS0rKy0rKystJSIiICIiIh8eICUiIygoKSkpKCkoKSgpJSUgHh4eHhsWFRUWFhcbGxYTFxsXFhsbHh4lIB9YJSVVWE8NUA0nKiolJyggIgIJCAhXKSUqLCwtLS0rLSsrLS0tLS0tLS0tLS0tLS0tLSwrKysrKy0sKyssLCsrLS0tKSkpKCgoKCgoKCUlJSUlHhsbGxsbFhYXFhUWFhUTEx4bGxsbGyAkJycbFRUXFhYVFRVPT09QT1VYHw1VDVgoKlFQUVVVTAYPDA9XJCUsKy0tLS0tLSstLCssKy0tLS0tLS0tLS0sKysrKysrLSorKysrKywtLS0rKCcjJR8fHhsbFhYVExMTExMTDxMQExMTFRUVFhsmKCgoKCgoKCIfHyAfICcnJycoIyIgJigoKSkiTRBPDSUqVVVYWFgfBghLDg8hJSorKy0tLSstKy0tKywsLS0tLS0tLS0tKysrKyssLSwrKysrKysrLS0tKystIB8fGxcXExMTExMTFRsbHh4iKCgoKCgoKCgoKCkpKCYmJyIiIiIgIh4WFhUeICAiKSkpKSkpKU5RTQwQTykoKCknJSIGBgMPDk8pKywoKy0tKywrKy0rKystLS0tLS0tLS0tLS0tLS0rKysrKysrKy0tLSwrKy0sExMTDxUVGxsfIigoKCgnJiAfFhcbGxsiIiIiHh4bFhUVFRUWFRYcGygoKCgoKCkpKCcoIiIiIx9VDVVRJSkiIh9PIygGBg8IECUpVSspLC0rLCsrLCsrKysrKysrKystLS0tLS0rKysrKysrKystLS0sKyssLSsqIyclJiIiIB4eHh4bGxsgIiIiIh4bFRUVFRUVFhcbGx8iIiopKSkpKSoqKSgpKSIlISAfVVVYUVgNDR9NKFVRUFYqKSoGDggQHypWKykrLSstKywtKywrKysrKy0tLSwrKyoqKysrKyoqKysrLS0rKysrLSwqKikqFRUVFRUWFhYWFhYWFhYWFRUVFhwfICIoKSkmKSkqKysrKioqJicmKCkjIiIgGxcWFhwgJygoKU8NWAxYKCoqKiokIgYFEAlPKVgqVS0tLCwrLSsrKywtLS0tLSsrKikoKCkpKSgoKSosLCwrKy0sLS0sKioqKSgoKCgoKCUfGxcWFRYWFhseICAlKSsrKiUlKCsqKioqKioqKikpKSkgHx8bGxcbHyApKSkoIh9YWBANUVAoKioqJSBYAQkPCE0oIk8hIS0sKystKystLSsqKSkoKCgoKCgoKSksLC0tLCsrKy0tLCoqKSkpKSUjKSgpGxsbFhYbHiAiKCoqKiorKyooIiUqKisqKysqKykpJh8gHx8fHh4bHx8iKCgiIyIgIR4bVVFQT1gMDSgpKiIiWFhMDw4ITSoiT1goLCcrLC0rLCwqKigpLC0oKCgoKCgpKSkpKSkrKywsLCsrKCkpKSkpKSopKCMpIB8iKCgrKyoqKiIiIiAgJSkqKysrKykoIh8fHyAgJSgoKCcjIiMkICIfGxYWFhsXFRYWUFUiI00QJiIqWFgfVVUODwhNKB9PHypPIystLSwqKyssKi0sLSgpKy0tLS0tLSwpKSkpKSkpKSkoKSgoKSciJSkjJCkoKysqKiklICAgICUpKikqKisqKSYgICAiICAeHiIeGxsbGxsbFhYWFhYWGx4bHBUWFyElKCkpKg8hVR9VVVBPTglLDE0mVU8gIk8hKy0sKy0tLS0tKykoKSspKSgoKCgmJiYmJiYpKikqLCspKCYmIigoKSgpKSIkIB8fHyAiKCkoKCgmKCIgIiIiIBsbGxsbHx8gICAgIiAgHxcVFhYeHyAXFh4iKCopKSoqKSghH1YNJVhQUAgJTw4MVVVRVU1PICstKywtLS0rKissLCwsLSwrKioqKioqKioqKiopJigoKCksLCsoJSUlKCgpKioqICEiICAiIiIeHx8eHh8bGx4fICAfHx8fFhYWFhYWFhYWFhslIR4VFhUeKSkqKikoKCIgICAfUBAiVU9LClBPDwhRIE1PTVcoKyEjKisrLS0tLS0tLS0tLCwsLCwsKikoKCgoKCgpKSksLCwqJigoKSwrKysrKyAgFxcXFxcbGx4eHx8fHxcXGxYWFhYWFhYWFhYWFhYbICAiKCAgIB8gIikpKSUgKCAiICIiIBcXUE0iTQpPVU0JCBAmTU8gJVVNT1csLS0tLS0tLS0tLSsrKCcpKSkqKioqKiosLCwsLCopKSkqKywrKysrLCkgICIoExMTExYXFhYbGxUWFhYWFhYWFhYWFhYeICAgHyIiICAgICAgICAoKCgiKCgmICAiIBsXFxYVHlYQVVVNCksOTU9NVQ9NCQ8iTlcpKystLSkjIyEhISEjISIoKioqKiorLCosLCwsLCwrLCwsLCwsKywrJiAgJSgoGxsbFhYWFxsXGx4WFhYWHyAiIiIiICIgHxsbHh4gICAgIiIiIyUlJCIiIBsbGxYWFRUeHhwdExknTQgIDBBPS0tLCgMICE0hKysrJyEfVyMhLCkrLC0rLCssLC0tLSwsLCwsLCwsKyssKywsJyIgICAfICIoKSIiGxscGxYWFhUWFRUWFhYWFhYWHh8lJiYmJSUlJSUgHhsbFhUWFhUWFhYeFRYVGxsbFg8PCQ8VFh4hTFVVUlNLS1IfIiglIiUiJSssLSwtLS0tLS0sKigoJiUlJSYoKCorKysqKCYfHyAfIB8gICkpKSkoIiImKSYoCgoKChMVFRUVFRsbICAfIiIiHxsbFRUVFhUVFRUVFRUWGxsbGxYWGxYWFhcbFRERFRcVFhUTCRolKCgpJioqJiglV1QpKisrKysrKyspJiYmKCgoKCoqKiopIiAiIiIiKCIgIiAiJSYrLCwsLCwrKSkoKygfHx8iERERERERERERERERDxEJCQkQFRUVFRUVHx8fHx4fHx8bFhUVFhYVEQkRDwgIDw8PDw8PCAkKCBEVGxseV1JMUgkJCgoJS0tNVCEqKywtLSwtLSwsKykpLC0qKikpKCIiIiIpKywsLC0sLCwsKyooKCAfHx8fIigqLi4uLi4uLgoKCgoKCgoKLi4uLi4uLi4ICgoKCgoKCgoKMDAwMjIyMDIJLi4uLi4uMzQ4ODg4MjQ7Pz89NDQ4MDAxLjUyN0FISEdDPz9CR0lJSkpJSUpISEhISUlJSUlJSUlJSUlHQ0NCQkI/PT0/Pz89PUJHR0dHMDAyMzQ0LjAuLi4uCQgICTAwMDAuLi4uLi4wLy4uLi8zMjIyMi4uLg8wMDAKCgoKCgoKCQouLg8zODw9PTs9PC49PTxFRUZIRkZJSUlKSkpKSkpKSkpJSUdDQkNDQ0VFRUZFQkJCQkVHR0dHR0hISEhISEdHR0ZGMjI0NDw9Pj09Pj4+Pj46NjExMTAPMDAwMDAwMDAKCgoKCgoKCS4uMDExMjIyMjIyCS4uCgoKCi40Ozs7ODgwLi4uMDExNjo/P0VDQ0NHSEdFRUREREhISEhGRUVGRkZHRkVGR0ZHR0ZGQkM/QkVDQz8/Pz8/Pz9FNDQ0NDQ2Njc6Oj9BQEBAQEBAQUFBQD4+OjY2NjMuLwoKCgoJCQkJCQkJCgkuMDM4ODg4NDQzMTE0NDk5Ozs7OTk3Nzk3OTk7PUJFRUREREVFRUVEPz08Pj09PT08PDMzMzMzMzg4O0JDQ0NDQ0NDRUVGRkVFQz89MDAwMDAwNDQ0NDk5Ojo6PEBARERBREFBQUFERDo6NzY0NjYxMTExMTExDw8PMDAyNDQ0NDAyMi4uMDQ0NDQ0NDQ4ODg4ODg5NDQ0NDQ0NDQ0NDQ0NjQ2Nzc+PT89PTs8Oz9DRUNCQ0NDQj08Ozk4OTk7OzxCRUNFCQgICS4PDzAwMDAxMTExMTY2OjxBQUFEQUREREREREREPjo6NzQ2MTExMzQzMjAwMDAwDzAwMg8wMDAyMjIwMDAuMDExMTE5ODk5OTk0MDAwMTEwMDIyMzM0NDQ5OTc+QEFAQEBAQEBARERBQkNDQ0NDQ0NDQ0I/Ly4uDw8PLi4uLi4uLi4uCS4uMTEuMTExNzc6OjxBRERERD8/NDIwLi4uLi4xNDY4ODg4ODg0NDQ0NDY2NDc4NDQ0Li4uLy4xNDk5ODg4OTw6Pj48PDk4ODg4ODQ0NjY2Ojo2MTExMTExMTE2Njs5PT08PENDQ0NDLy8uLi4uLi4uLi4uLi4JCQkJCQkuLi8uLi4uMTE2Ojo6Ojo5NzQ0NDExMTIzMzQzODg4ODg4ODQ2NjY2Njk6Ojw6PDo6Ojo6PDs9PT09P0NDRUNDPz09PTw8Ozw7Ojo6Ojo6Ojo6Ojo6OjY2NDQ0ODg4PDw8PT09Li4uLy8vLy8vCQkJCQkJCQkJCQkJCQkJLi4vLy4xNjY3Nzc3NDQ0NDQ0NDQ2NDM0NDQ4ODg6Ojk5OTk5OTk7Pj5BQUE9Pj4+Pj09P0NDQ0NDQ0NDQ0NCPT07OTg4OTk5Ojo8Pz8/Pzw8OTk6PDw9Pz89PDw8PDw8CQkJCS8vLy8vLy8vLy8vCQkJCQkJCS8vLzE0NDExMTQ0MzM0NDc3OTk5OTk5OTk5OTc3Nzc3Nzw/Pz8/Pz8/PT4/RUVFRUVFRUVFRURDQ0NCQkI/Pzw8PDk4ODg4ODg4PT06OTo7OTg4ODk5OTk5OTo6ODg4Nzg3Ly8vCQkJCQkvLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vMTQ0NDQ0NDQ5OTk5OTk0MzMzMTExMTE0NDk5OTk7OTk9Pz9ERUVFRUVFQ0NDPz8/PDw7OTg5PDw8PDw/PT89PT09Qj9CQkI/Pz06OTYxNjExNjY2Nzc3Ly8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy4uMzMyMzMzMzMzMzEvLy8vLy8vMTQ0NDQ0ODk4ODw8PDw6Ojo5Ojk6Ojo6ODg4ODg4ODg4PDw/PT8/Q0NDQ0NDQ0NDQ0NAQEA+Ojo6PDg3Nzc4Ly8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vLy8vNTUvLy8vLy8xNDY2NjY5OTk5NDQzNDMzMTExLy8vLy8xMTE0NDQ0ODg4ODQ0NDQ0NDQ0NDQ0NDQzMzQ0Njg4OTs5PD0/Pz89Q0NDQ0NDQ0VFREREREA+Pjw8PDs7NjY2NjY2NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1NTU1MTY2NjY5Ojo5OTk5OTk3NjYxMTExMTExMTY2Njc3Nzc4ODg0NDQxMTExMDQ0NDQ0Njc3Ojo6Ojo6Nzk5OTk5OT09PT0/Pz8+Pj4+REREREBCQkJCPT09NjY2NjY2NTU1NTU1NTU1NTExNTU1NTU1NTU1NTU1NTU1NjY2NjY2NjY2NjY2NjYxMS8vLy8xNjY3Nzk6Ojc3Nzg0NDQ0NDQ0NDQ0NDY2NDQ0NjY2MTExNjY2NjY3Nzc3Nzc6Ojo8PDo6Oj4+Pj4+Pj49QkJCQkI9NjY2NjY2NjY1NTU1NTU1NjY2NjYxNTUxMTY2NjY3Nzc3Nzc3NzY0NDY2MTExLy8vLy8vLy8xNjc3Nzc2NDY0NDQ0NDQ0NDc3Nzc2NjY2NjY2NjY2NjY2NjY2NjY2Nzc3Nzc6Ojo6Ojo6Ojo6Pj4+PT09PUJCQkJCPj4+Pjo2NjY2NjY3Nzc3Nzc2NjY2NjY2Ojo6Ojo6Ojo6Ojo6Ojo6OTk2NjExMTExMTE2NjY2NjY2NjY2NjY2NjQ0NDQ0NDM0NDQ0NjY3Ojc3NjY6Ojo3Nzc3Nzc3Nzc3Nzo3Ojo6Ojo6Ojo5OTw8PD0/P0JCQkJCPj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj46Ojo6Ojo6OjY2NjY2NjY2NjY2NjY2Ojo6Ojo6OTk6OTk6Ojo8Ojw6Ojo6NjY2NjY2NjY2NjY2Njc3Nzc3NzY2NjY2NjY2Njo5OTk9PT09QkJCQkJCQUBBQUFAQEFAPj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pjk6Ojo5OTo3NjY2MTExMTY2Njo6Ojo6OTo5Ojo6PT09PT09PT06Ojo6NjY2NjY2NjY2NjY3Nzc3Nzc3NzY2NjY2Njc5OTk6PT0/QkJCQkJCQUFBQEFBQEBAQEFBQUFBQUFBQUBBQUFBQUFBQUE+Pj4+Pj4+Pj4+Pj46Ojo6NjY2NjY2Njo3Nzc2NjY6Ojo6Ojo6PT09PT89PT09PT09Pj46Ojo3NjY2NjY3Nzc3Nzc3Nzc3Nzc6Nzc2Njc6Ojk5Oj4+QkJCQkJCQUFBQUFBQUFAQUFBQUBAQUFBQEFBQUBAQEA+Pj4+Pj4+Pj4+Pj4+Ojo6Ojo6Ojo6Ojo6Ojo6Ojo6Ojo6Ojo6Oj09PUJCQkJCQkJCQj89PDo6Ojo6Ojo6Ojo6Ojo6Nzc3Ojo3Nzc6Nzc3Nzc3Nzo6Pj4+Pj4+Pj4+QUFAQEBBQUFBQUFBQUBBQUFAQEFBQUFBQUBAQD4+Pj4+Pj4+Ojo6Ojo6Ojo6Ojo6Ojo6Ojo8PD49Pz8+P0I+Pj0/PT1CQkJCQkJCQkI9PT4+Pz8/PDo6Ojo6Ojo6Ojo+Pj46Ojc3Nzc3Nzc3Nzo+Pj4+Pj4+Pj4+QUFBQUFBQUFBQUFBQEFBQUFBQUFAQUFBQUFAQEE+Pj4+Pj4+Pjo6Ojo6Ojo6Ojo6Ojo+Pj4+Pj0+PkJCQkJCQkJCQkJCQkJCQkJCQkJCQkJCPz8/Pz4+Pj46Ojo9Pj4+Pj46Ojo3Nzc3Nz4+Pj4+Pj4+Pj4+Pj4+QUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUBBQUFBQT4+Pj4+Pj4+Ojo6Ojo6Ojo6Ojo6Oj4+Pj4+Pj4+QkJCQkJCQkJCQkJCQkJCQkJCQ0JCQkJCQj8+Pj4+Pj4+Pj4+Pj4+Pj46Ojo6Nzc3Pj4+Pj4+Pj4+Pj4+QEFBQUFBQUFBQUFBQUBBQUFBQEBAQEBBQUFAQD4+Pj4+Pj4+Pj4+Pj4+Pjo6Ojo6Ojo6Ojo6Ojo6Pj4+Pj9CQkJCQkJCQkJCQkJCQkJCQkNCQ0JCQkJDQkI+Pj4+Pj4/QkI+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+QUBBQUFBQUFBQUFBQUFBQUFBQEBBQUFBQUFAQEFBQUE+Pj4+Pj4+Pj4+Pj46Ojo6Ojo6Ojo6Oj0+PkJCQkJCQkJCQkJCQkJCQkJCQkJCQ0NCQ0NCQkJDQkNDQkNCQkJCQkI+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+Pj4+";

const UI_PRESETS = {
  white: { ui: 0x7FFF, selected: 0x037F },
  cyan: { ui: 0x7FE0, selected: 0x7FFF },
  blue: { ui: 0x7C00, selected: 0x7FE0 },
  yellow: { ui: 0x037F, selected: 0x7FFF },
  green: { ui: 0x03E0, selected: 0x7FFF },
  pink: { ui: 0x7C1F, selected: 0x7FFF },
  orange: { ui: 0x01FF, selected: 0x7FFF }
};
const OUTLINE_PRESETS = { black:0x0000, navy:0x2400, white:0x7FFF, blue:0x7C00, yellow:0x037F };
const COMMON_GBA_COLORS = Object.freeze([
  ['White',0x7FFF], ['Gray',0x4210], ['Black',0x0000], ['Red',0x001F], ['Orange',0x021F],
  ['Yellow',0x037F], ['Green',0x03E0], ['Cyan',0x7FE0], ['Blue',0x7C00], ['Magenta',0x7C1F]
].map(([name,rgb15])=>Object.freeze({name,rgb15,hex:rgb555ToHex(rgb15)})));
let activeGBAColorPicker = null;

function fromBase64(value) {
  const binary = atob(value); const out = new Uint8Array(binary.length);
  for (let i=0;i<binary.length;i++) out[i]=binary.charCodeAt(i); return out;
}
function toBase64(bytes) {
  let result=''; const block=0x8000;
  for (let i=0;i<bytes.length;i+=block) result += String.fromCharCode(...bytes.subarray(i,i+block));
  return btoa(result);
}
function setColour(palette, index, value) { palette[index*2]=value&255; palette[index*2+1]=(value>>>8)&255; }
function getColour(palette, index) { return palette[index*2] | (palette[index*2+1]<<8); }
function rgb555ToRGB(value) { return [Math.round((value&31)*255/31),Math.round(((value>>>5)&31)*255/31),Math.round(((value>>>10)&31)*255/31)]; }
function rgb555(r,g,b) { return Math.round(r*31/255) | (Math.round(g*31/255)<<5) | (Math.round(b*31/255)<<10); }
function rgb555ToHex(value) {
  const [r,g,b]=rgb555ToRGB(value&0x7FFF);
  return `#${[r,g,b].map(channel=>channel.toString(16).padStart(2,'0')).join('')}`.toUpperCase();
}
function hexToRGB555(value,fallback=0) {
  if(typeof value==='number'&&Number.isFinite(value)) return value&0x7FFF;
  const normalized=normalizeHexColor(value);
  if(!normalized) return fallback&0x7FFF;
  const number=parseInt(normalized.slice(1),16);
  return rgb555((number>>>16)&255,(number>>>8)&255,number&255);
}
function colourSetting(value,fallback,presets={}) {
  if(typeof value==='number'&&Number.isFinite(value)) return value&0x7FFF;
  if(typeof value==='string'&&Object.prototype.hasOwnProperty.call(presets,value)) return presets[value]&0x7FFF;
  return hexToRGB555(value,fallback);
}
function quantizeHexColor(value,fallback='#000000') { return rgb555ToHex(hexToRGB555(value,hexToRGB555(fallback,0))); }
function describeColor(value,fallback='#000000') {
  const rgb15=hexToRGB555(value,hexToRGB555(fallback,0));
  return {rgb15,hex:rgb555ToHex(rgb15),r:rgb15&31,g:(rgb15>>>5)&31,b:(rgb15>>>10)&31};
}

function normalizeHexColor(value) {
  let text=String(value??'').trim().replace(/^0x/i,'').replace(/^#/,'');
  if(/^[0-9a-f]{3}$/i.test(text)) text=text.split('').map(char=>char+char).join('');
  if(!/^[0-9a-f]{6}$/i.test(text)) return null;
  return `#${text.toUpperCase()}`;
}
function rgbToHSV(r,g,b) {
  const red=Math.min(255,Math.max(0,Number(r)||0))/255;
  const green=Math.min(255,Math.max(0,Number(g)||0))/255;
  const blue=Math.min(255,Math.max(0,Number(b)||0))/255;
  const maximum=Math.max(red,green,blue), minimum=Math.min(red,green,blue), difference=maximum-minimum;
  let hue=0;
  if(difference) {
    if(maximum===red) hue=60*(((green-blue)/difference)%6);
    else if(maximum===green) hue=60*((blue-red)/difference+2);
    else hue=60*((red-green)/difference+4);
  }
  if(hue<0) hue+=360;
  return {h:hue,s:maximum?difference/maximum:0,v:maximum};
}
function hsvToHex(h,s,v) {
  const hue=((Number(h)||0)%360+360)%360;
  const saturation=Math.min(1,Math.max(0,Number(s)||0));
  const value=Math.min(1,Math.max(0,Number(v)||0));
  const chroma=value*saturation;
  const section=hue/60;
  const secondary=chroma*(1-Math.abs(section%2-1));
  let red=0,green=0,blue=0;
  if(section<1) [red,green,blue]=[chroma,secondary,0];
  else if(section<2) [red,green,blue]=[secondary,chroma,0];
  else if(section<3) [red,green,blue]=[0,chroma,secondary];
  else if(section<4) [red,green,blue]=[0,secondary,chroma];
  else if(section<5) [red,green,blue]=[secondary,0,chroma];
  else [red,green,blue]=[chroma,0,secondary];
  const match=value-chroma;
  return `#${[red,green,blue].map(channel=>Math.round((channel+match)*255).toString(16).padStart(2,'0')).join('')}`.toUpperCase();
}
function setupGBAColorPicker(input,{label='GBA colour'}={}) {
  if(!input) return null;
  if(input._gbaColorPickerController) return input._gbaColorPickerController;
  const control=input.closest('.gba-color-control');
  if(!control || typeof document==='undefined') return null;
  control.classList.add('gba-color-enhanced');

  const trigger=document.createElement('button');
  trigger.type='button';
  trigger.className='gba-color-trigger';
  trigger.id=`${input.id}PickerButton`;
  trigger.setAttribute('aria-haspopup','dialog');
  trigger.setAttribute('aria-expanded','false');
  trigger.innerHTML='<span class="gba-color-trigger-swatch" aria-hidden="true"></span>';
  control.insertBefore(trigger,input);

  for(const externalLabel of document.querySelectorAll(`label[for="${input.id}"]`)) externalLabel.htmlFor=trigger.id;

  input.classList.add('gba-native-color-input');
  input.tabIndex=-1;
  input.setAttribute('aria-hidden','true');

  const panel=document.createElement('div');
  panel.className='gba-color-popover';
  panel.id=`${input.id}PickerPanel`;
  panel.hidden=true;
  panel.setAttribute('role','dialog');
  panel.setAttribute('aria-label',`${label} picker`);
  trigger.setAttribute('aria-controls',panel.id);

  const content=document.createElement('div');
  content.className='gba-picker-content';

  const svArea=document.createElement('div');
  svArea.className='gba-sv-area';
  svArea.tabIndex=0;
  svArea.setAttribute('role','slider');
  svArea.setAttribute('aria-label',`${label} saturation and brightness`);
  svArea.setAttribute('aria-valuemin','0');
  svArea.setAttribute('aria-valuemax','100');
  const svThumb=document.createElement('span');
  svThumb.className='gba-sv-thumb';
  svThumb.setAttribute('aria-hidden','true');
  svArea.append(svThumb);
  content.append(svArea);

  const pickerRow=document.createElement('div');
  pickerRow.className='gba-picker-row';

  const eyedropperButton=document.createElement('button');
  eyedropperButton.type='button';
  eyedropperButton.className='gba-eyedropper';
  eyedropperButton.setAttribute('aria-label',`Pick ${label.toLowerCase()} from the screen`);
  eyedropperButton.title='Pick a colour from the screen';
  eyedropperButton.innerHTML='<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18.5 2.5a2.12 2.12 0 0 1 3 3L13 14l-3-3 8.5-8.5Z"/><path d="m9 12-5.2 5.2a2 2 0 0 0-.6 1.4V21h2.4a2 2 0 0 0 1.4-.6L12 15"/><path d="m14.5 6.5 3 3"/></svg>';

  const currentSwatch=document.createElement('span');
  currentSwatch.className='gba-current-swatch';
  currentSwatch.setAttribute('role','img');
  currentSwatch.setAttribute('aria-label',`${label} preview`);

  const hueSlider=document.createElement('input');
  hueSlider.type='range';
  hueSlider.min='0';
  hueSlider.max='359';
  hueSlider.step='1';
  hueSlider.value='0';
  hueSlider.className='gba-hue-slider';
  hueSlider.setAttribute('aria-label',`${label} hue`);
  hueSlider.title='Hue';
  pickerRow.append(eyedropperButton,currentSwatch,hueSlider);
  content.append(pickerRow);

  const rgbEditor=document.createElement('div');
  rgbEditor.className='gba-rgb-editor';
  const rgbInputs=[];
  for(const channel of ['R','G','B']) {
    const channelLabel=document.createElement('label');
    channelLabel.className='gba-rgb-channel';
    const caption=document.createElement('span');
    caption.textContent=channel;
    const numberInput=document.createElement('input');
    numberInput.type='number';
    numberInput.min='0';
    numberInput.max='255';
    numberInput.step='1';
    numberInput.inputMode='numeric';
    numberInput.setAttribute('aria-label',`${label} ${channel}`);
    channelLabel.append(caption,numberInput);
    rgbEditor.append(channelLabel);
    rgbInputs.push(numberInput);
  }
  content.append(rgbEditor);

  const hexLabel=document.createElement('label');
  hexLabel.className='gba-hex-editor';
  const hexCaption=document.createElement('span');
  hexCaption.textContent='HEX';
  const hexInput=document.createElement('input');
  hexInput.type='text';
  hexInput.inputMode='text';
  hexInput.autocomplete='off';
  hexInput.spellcheck=false;
  hexInput.maxLength=9;
  hexInput.placeholder='#RRGGBB';
  hexInput.setAttribute('aria-label',`${label} HEX code`);
  hexLabel.append(hexCaption,hexInput);
  content.append(hexLabel);

  const quickGrid=document.createElement('div');
  quickGrid.className='gba-quick-colors';
  const quickButtons=[];
  for(const colour of COMMON_GBA_COLORS) {
    const button=document.createElement('button');
    button.type='button';
    button.className='gba-quick-color';
    button.style.background=colour.hex;
    button.title=`${colour.name} — ${colour.hex}`;
    button.setAttribute('aria-label',`Choose ${colour.name} ${colour.hex}`);
    button.dataset.hex=colour.hex;
    quickGrid.append(button);
    quickButtons.push(button);
  }
  content.append(quickGrid);
  panel.append(content);
  control.append(panel);

  const swatch=trigger.querySelector('.gba-color-trigger-swatch');
  const EyeDropperClass=typeof window!=='undefined'?window.EyeDropper:null;
  let lastHue=0;
  let saturation=0;
  let brightness=1;
  let draggingSV=false;

  function clamp01(value) { return Math.min(1,Math.max(0,value)); }
  function close() {
    panel.hidden=true;
    trigger.setAttribute('aria-expanded','false');
    if(activeGBAColorPicker===controller) activeGBAColorPicker=null;
  }
  function open() {
    if(input.disabled) return;
    if(activeGBAColorPicker && activeGBAColorPicker!==controller) activeGBAColorPicker.close();
    panel.hidden=false;
    trigger.setAttribute('aria-expanded','true');
    activeGBAColorPicker=controller;
    panel.classList.remove('align-right');
    const bounds=panel.getBoundingClientRect();
    if(bounds.right>window.innerWidth-10) panel.classList.add('align-right');
  }
  function dispatch(kind) { input.dispatchEvent(new Event(kind,{bubbles:true})); }
  function setRawHex(hex,commit=false) {
    const normalized=normalizeHexColor(hex);
    if(!normalized) return false;
    input.value=normalized;
    dispatch('input');
    if(commit) dispatch('change');
    return true;
  }
  function applyRGB(commit=false) {
    const values=rgbInputs.map(field=>Number(field.value));
    if(values.some(value=>!Number.isFinite(value))) return;
    const clamped=values.map(value=>Math.min(255,Math.max(0,Math.round(value))));
    const hex=`#${clamped.map(value=>value.toString(16).padStart(2,'0')).join('')}`;
    setRawHex(hex,commit);
  }
  function applyHue(commit=false) {
    const hue=Math.min(359,Math.max(0,Math.round(Number(hueSlider.value)||0)));
    lastHue=hue;
    setRawHex(hsvToHex(lastHue,saturation>0.01?saturation:1,brightness>0.01?brightness:1),commit);
  }
  function applySV(clientX,clientY,commit=false) {
    const bounds=svArea.getBoundingClientRect();
    if(bounds.width<=0||bounds.height<=0) return;
    saturation=clamp01((clientX-bounds.left)/bounds.width);
    brightness=clamp01(1-(clientY-bounds.top)/bounds.height);
    setRawHex(hsvToHex(lastHue,saturation,brightness),commit);
  }
  function sync() {
    const colour=describeColor(input.value,'#000000');
    trigger.style.setProperty('--gba-swatch-color',colour.hex);
    swatch.style.backgroundColor=colour.hex;
    currentSwatch.style.backgroundColor=colour.hex;
    trigger.setAttribute('aria-label',`${label}: ${colour.hex}. Open colour picker`);
    currentSwatch.setAttribute('aria-label',`${label} preview ${colour.hex}`);
    const [r8,g8,b8]=rgb555ToRGB(colour.rgb15);
    for(const [field,value] of rgbInputs.map((field,index)=>[field,[r8,g8,b8][index]])) {
      if(document.activeElement!==field) field.value=String(value);
    }
    const hsv=rgbToHSV(r8,g8,b8);
    saturation=hsv.s;
    brightness=hsv.v;
    if(hsv.s>0.01&&hsv.v>0.01) lastHue=hsv.h;
    const displayedHue=Math.round(lastHue)%360;
    if(document.activeElement!==hueSlider) hueSlider.value=String(displayedHue);
    const hueColour=`hsl(${displayedHue} 100% 50%)`;
    hueSlider.style.setProperty('--hue-colour',hueColour);
    svArea.style.setProperty('--hue-colour',hueColour);
    svArea.style.setProperty('--sv-x',`${saturation*100}%`);
    svArea.style.setProperty('--sv-y',`${(1-brightness)*100}%`);
    svArea.setAttribute('aria-valuenow',String(Math.round(brightness*100)));
    svArea.setAttribute('aria-valuetext',`${Math.round(saturation*100)}% saturation, ${Math.round(brightness*100)}% brightness`);
    if(document.activeElement!==hexInput) hexInput.value=colour.hex;
    hexInput.classList.remove('invalid');
    for(const button of quickButtons) button.setAttribute('aria-pressed',String(button.dataset.hex===colour.hex));
    const disabled=Boolean(input.disabled);
    trigger.disabled=disabled;
    hueSlider.disabled=disabled;
    svArea.tabIndex=disabled?-1:0;
    svArea.setAttribute('aria-disabled',String(disabled));
    for(const field of rgbInputs) field.disabled=disabled;
    hexInput.disabled=disabled;
    for(const button of quickButtons) button.disabled=disabled;
    eyedropperButton.disabled=disabled||typeof EyeDropperClass!=='function';
    eyedropperButton.title=typeof EyeDropperClass==='function'?'Pick a colour from the screen':'Screen eyedropper is not available here';
    if(disabled) close();
  }

  trigger.addEventListener('click',event=>{event.preventDefault();panel.hidden?open():close();});
  svArea.addEventListener('pointerdown',event=>{
    if(input.disabled) return;
    draggingSV=true;
    svArea.setPointerCapture?.(event.pointerId);
    applySV(event.clientX,event.clientY,false);
    event.preventDefault();
  });
  svArea.addEventListener('pointermove',event=>{
    if(!draggingSV||input.disabled) return;
    applySV(event.clientX,event.clientY,false);
  });
  const finishSV=event=>{
    if(!draggingSV) return;
    draggingSV=false;
    applySV(event.clientX,event.clientY,true);
    svArea.releasePointerCapture?.(event.pointerId);
  };
  svArea.addEventListener('pointerup',finishSV);
  svArea.addEventListener('pointercancel',event=>{
    draggingSV=false;
    svArea.releasePointerCapture?.(event.pointerId);
  });
  svArea.addEventListener('keydown',event=>{
    if(input.disabled) return;
    const step=event.shiftKey?0.10:0.02;
    if(event.key==='ArrowLeft') saturation=clamp01(saturation-step);
    else if(event.key==='ArrowRight') saturation=clamp01(saturation+step);
    else if(event.key==='ArrowUp') brightness=clamp01(brightness+step);
    else if(event.key==='ArrowDown') brightness=clamp01(brightness-step);
    else return;
    event.preventDefault();
    setRawHex(hsvToHex(lastHue,saturation,brightness),true);
  });
  for(const field of rgbInputs) {
    field.addEventListener('input',()=>applyRGB(false));
    field.addEventListener('change',()=>applyRGB(true));
    field.addEventListener('blur',()=>setTimeout(sync,0));
    field.addEventListener('keydown',event=>{if(event.key==='Enter'){event.preventDefault();applyRGB(true);field.select();}});
  }
  hexInput.addEventListener('input',()=>{
    const normalized=normalizeHexColor(hexInput.value);
    hexInput.classList.toggle('invalid',!normalized);
    if(normalized) setRawHex(normalized,false);
  });
  const commitHex=()=>{
    const normalized=normalizeHexColor(hexInput.value);
    if(!normalized){hexInput.classList.remove('invalid');sync();return;}
    setRawHex(normalized,true);
    sync();
  };
  hexInput.addEventListener('change',commitHex);
  hexInput.addEventListener('blur',()=>setTimeout(sync,0));
  hexInput.addEventListener('keydown',event=>{if(event.key==='Enter'){event.preventDefault();commitHex();hexInput.select();}});
  hueSlider.addEventListener('input',()=>applyHue(false));
  hueSlider.addEventListener('change',()=>applyHue(true));
  hueSlider.addEventListener('blur',()=>setTimeout(sync,0));
  eyedropperButton.addEventListener('click',async()=>{
    if(typeof EyeDropperClass!=='function'||input.disabled) return;
    try {
      const result=await new EyeDropperClass().open();
      if(result?.sRGBHex) setRawHex(result.sRGBHex,true);
    } catch(error) {
      if(error?.name!=='AbortError') console.warn('Screen eyedropper failed',error);
    }
  });
  for(const button of quickButtons) button.addEventListener('click',()=>setRawHex(button.dataset.hex,true));
  input.addEventListener('input',sync);
  input.addEventListener('change',sync);
  document.addEventListener('pointerdown',event=>{if(!panel.hidden&&!control.contains(event.target)) close();});
  document.addEventListener('keydown',event=>{if(event.key==='Escape'&&!panel.hidden){close();trigger.focus();}});

  const controller={open,close,sync,panel,trigger};
  input._gbaColorPickerController=controller;
  if(typeof MutationObserver!=='undefined') new MutationObserver(sync).observe(input,{attributes:true,attributeFilter:['disabled']});
  sync();
  return controller;
}
function settingsColours(settings={}) {
  const pair=UI_PRESETS[settings.uiColor] || UI_PRESETS.white;
  const uiPresets=Object.fromEntries(Object.entries(UI_PRESETS).map(([name,colours])=>[name,colours.ui]));
  const selectedPresets=Object.fromEntries(Object.entries(UI_PRESETS).map(([name,colours])=>[name,colours.selected]));
  return {
    ui:colourSetting(settings.uiColor,pair.ui,uiPresets),
    selected:colourSetting(settings.selectedColor,pair.selected,selectedPresets),
    outline:colourSetting(settings.outlineColor,OUTLINE_PRESETS.black,OUTLINE_PRESETS)
  };
}
function applyUI(theme, settings={}) {
  const c=settingsColours(settings); theme.uiColor=c.ui; theme.selectedColor=c.selected; theme.outline=!!settings.outline; theme.outlineColor=c.outline; return theme;
}
function makePalette(values) { const p=new Uint8Array(512); values.forEach((v,i)=>setColour(p,i,v)); return p; }
function classicTheme(settings={}) {
  const palette=makePalette([0x0000,0x0800,0x1000,0x1800,0x2000,0x2820,0x3040,0x3860]);
  const frame=new Uint8Array(FRAME_BYTES);
  for(let y=0;y<FRAME_HEIGHT;y++) for(let x=0;x<FRAME_WIDTH;x++) {
    const band=Math.min(7,Math.floor(y/10)); frame[y*FRAME_WIDTH+x]=Math.max(1,7-band);
    if(y>54 && ((x+y)&7)===0) frame[y*FRAME_WIDTH+x]=Math.min(7,frame[y*FRAME_WIDTH+x]+1);
  }
  return applyUI({id:'classic-dark',name:'Classic dark',kind:'static',palette,frames:[frame],frameVBlanks:0},settings);
}
function oceanTheme(animated,settings={}) {
  const theme={id:animated?'ocean-wave-animated':'ocean-wave-static',name:animated?'Ocean Wave — animated':'Ocean Wave — static',kind:animated?'palette-shimmer':'static',palette:fromBase64(OCEAN_PALETTE_B64),frames:[fromBase64(OCEAN_PIXELS_B64)],frameVBlanks:0};
  if(animated) theme.shimmer={sourceStart:17,count:29,target1:46,interval1:12,target2:75,interval2:30,phases:4};
  return applyUI(theme,settings);
}
function blueWaveTheme(settings={}) {
  const palette=new Uint8Array(512);
  const colors=[];
  for(let i=0;i<16;i++) colors.push(rgb555(10+i*3,35+i*5,90+i*8));
  for(let i=0;i<16;i++) colors.push(rgb555(5+i*2,50+i*6,125+i*7));
  for(let i=0;i<8;i++) colors.push(rgb555(90+i*20,170+i*10,230+i*3));
  colors.forEach((c,i)=>setColour(palette,i,c));
  const frames=[]; const count=12;
  for(let f=0;f<count;f++) {
    const frame=new Uint8Array(FRAME_BYTES); const phase=f*Math.PI*2/count;
    for(let y=0;y<FRAME_HEIGHT;y++) for(let x=0;x<FRAME_WIDTH;x++) {
      const horizon=43 + Math.sin(x/13+phase)*3 + Math.sin(x/27-phase*1.4)*2;
      let idx;
      if(y<horizon) idx=Math.max(0,Math.min(15,Math.floor((y/Math.max(1,horizon))*15)));
      else {
        const depth=y-horizon; const ripple=Math.sin(x/8+phase*2+depth/5)+Math.sin(x/19-phase+depth/9);
        idx=16+Math.max(0,Math.min(15,Math.floor(depth/3+ripple*1.5)));
        if(Math.abs(depth)<1.5 || (depth<8 && ripple>1.65)) idx=32+((x+f*3)&7);
      }
      frame[y*FRAME_WIDTH+x]=idx;
    }
    frames.push(frame);
  }
  return applyUI({id:'blue-wave-animated',name:'Blue Wave — animated',kind:'frames',palette,frames,frameVBlanks:12},settings);
}
function createBuiltinTheme(id,settings={}) {
  if(id==='ocean-wave-static') return oceanTheme(false,settings);
  if(id==='ocean-wave-animated') return oceanTheme(true,settings);
  if(id==='blue-wave-animated') return blueWaveTheme(settings);
  return classicTheme(settings);
}
function buildCubePalette() {
  const p=new Uint8Array(512), levels=[0,51,102,153,204,255]; let index=0;
  for(let r=0;r<6;r++) for(let g=0;g<6;g++) for(let b=0;b<6;b++) setColour(p,index++,rgb555(levels[r],levels[g],levels[b]));
  return p;
}
function quantizeRGBA(data) {
  const out=new Uint8Array(FRAME_BYTES);
  for(let i=0;i<FRAME_BYTES;i++) {
    const p=i*4; const r=Math.min(5,Math.round(data[p]/51)); const g=Math.min(5,Math.round(data[p+1]/51)); const b=Math.min(5,Math.round(data[p+2]/51)); out[i]=r*36+g*6+b;
  }
  return out;
}
function quantizeRGB24(data,offset=0) {
  const out=new Uint8Array(FRAME_BYTES);
  for(let i=0;i<FRAME_BYTES;i++) {
    const p=offset+i*3; const r=Math.min(5,Math.round(data[p]/51)); const g=Math.min(5,Math.round(data[p+1]/51)); const b=Math.min(5,Math.round(data[p+2]/51)); out[i]=r*36+g*6+b;
  }
  return out;
}
function isVideoFile(file) {
  return !!file && ((typeof file.type==='string' && file.type.startsWith('video/')) || /\.(mp4|m4v|mov|webm|mkv|avi|mpg|mpeg)$/i.test(String(file.name||'')));
}
function decodeRGB24Frames(data,name,frameVBlanks,settings={},progress=()=>{}) {
  const bytes=data instanceof Uint8Array?data:new Uint8Array(data||0);
  if(bytes.length<RGB24_FRAME_BYTES || bytes.length%RGB24_FRAME_BYTES!==0) throw new Error('Video decoder returned incomplete menu frames.');
  const total=Math.min(16,Math.floor(bytes.length/RGB24_FRAME_BYTES));
  const palette=buildCubePalette(),frames=[];
  for(let i=0;i<total;i++) { frames.push(quantizeRGB24(bytes,i*RGB24_FRAME_BYTES)); progress((i+1)/total); }
  const interval=frames.length>1?Math.max(6,Math.min(120,Math.round(Number(frameVBlanks)||12))):0;
  return applyUI({id:'custom',name:name||'Custom video',kind:frames.length>1?'frames':'static',palette,frames,frameVBlanks:interval},settings);
}
function drawBitmapToRGBA(bitmap) {
  const canvas=document.createElement('canvas'); canvas.width=FRAME_WIDTH; canvas.height=FRAME_HEIGHT; const ctx=canvas.getContext('2d',{alpha:false});
  ctx.imageSmoothingEnabled=true; const bw=bitmap.displayWidth||bitmap.codedWidth||bitmap.width, bh=bitmap.displayHeight||bitmap.codedHeight||bitmap.height; const scale=Math.max(FRAME_WIDTH/bw,FRAME_HEIGHT/bh); const w=bw*scale,h=bh*scale;
  ctx.fillStyle='#000'; ctx.fillRect(0,0,FRAME_WIDTH,FRAME_HEIGHT); ctx.drawImage(bitmap,(FRAME_WIDTH-w)/2,(FRAME_HEIGHT-h)/2,w,h);
  return ctx.getImageData(0,0,FRAME_WIDTH,FRAME_HEIGHT).data;
}
async function decodeCustomFile(file,settings={},progress=()=>{}) {
  const palette=buildCubePalette(); const frames=[]; let frameVBlanks=12;
  const isGIF=(file.type==='image/gif'||/\.gif$/i.test(file.name));
  if(isGIF && typeof ImageDecoder!=='undefined') {
    const data=await file.arrayBuffer(); const decoder=new ImageDecoder({data,type:'image/gif'}); await decoder.tracks.ready;
    const total=Math.max(1,decoder.tracks.selectedTrack?.frameCount||1); const wanted=Math.min(16,total); const step=total/wanted;
    for(let i=0;i<wanted;i++) {
      const frameIndex=Math.min(total-1,Math.floor(i*step)); const decoded=await decoder.decode({frameIndex,completeFramesOnly:true});
      frames.push(quantizeRGBA(drawBitmapToRGBA(decoded.image))); decoded.image.close?.(); progress((i+1)/wanted);
    }
  } else {
    const bitmap=await createImageBitmap(file); frames.push(quantizeRGBA(drawBitmapToRGBA(bitmap))); bitmap.close?.(); progress(1);
  }
  const theme={id:'custom',name:file.name,kind:frames.length>1?'frames':'static',palette,frames,frameVBlanks:frames.length>1?frameVBlanks:0};
  return applyUI(theme,settings);
}
function serializeTheme(theme) {
  if(!theme) return null;
  return {id:theme.id||'custom',name:theme.name||'',kind:theme.kind,palette:toBase64(theme.palette),frames:theme.frames.map(toBase64),frameVBlanks:theme.frameVBlanks||0,uiColor:Number.isInteger(theme.uiColor)?theme.uiColor:0x7FFF,selectedColor:Number.isInteger(theme.selectedColor)?theme.selectedColor:0x037F,outline:!!theme.outline,outlineColor:Number.isInteger(theme.outlineColor)?theme.outlineColor:0,shimmer:theme.shimmer||null};
}
function deserializeTheme(value) {
  if(!value) return null; return {...value,palette:typeof value.palette==='string'?fromBase64(value.palette):value.palette,frames:(value.frames||[]).map(v=>typeof v==='string'?fromBase64(v):v)};
}
function paletteForPreview(theme,elapsed) {
  const p=theme.palette.slice(); if(theme.kind!=='palette-shimmer'||!theme.shimmer) return p;
  const s=theme.shimmer; const apply=(target,interval)=>{ const phase=Math.floor(elapsed/(interval/59.7275*1000))%s.phases; for(let i=0;i<s.count;i++) { let c=getColour(theme.palette,s.sourceStart+i); if(((s.sourceStart+i+phase)&(s.phases-1))===0) c=(c+0x0420)&0x7FFF; setColour(p,target+i,c); } };
  apply(s.target1,s.interval1); apply(s.target2,s.interval2); return p;
}
function drawGlyph(buffer,x,y,ch,fg,outline,outlineEnabled) {
  const bits=glyphBits(ch); const points=[];
  for(let row=0;row<5;row++) for(let col=0;col<3;col++) if(bits&(1<<(14-row*3-col))) points.push([x+col,y+row]);
  if(outlineEnabled) for(const [px,py] of points) for(let oy=-1;oy<=1;oy++) for(let ox=-1;ox<=1;ox++) { const xx=px+ox,yy=py+oy;if(xx>=0&&yy>=0&&xx<FRAME_WIDTH&&yy<FRAME_HEIGHT) buffer[yy*FRAME_WIDTH+xx]=outline; }
  for(const [px,py] of points) if(px>=0&&py>=0&&px<FRAME_WIDTH&&py<FRAME_HEIGHT) buffer[py*FRAME_WIDTH+px]=fg;
}
function drawText(buffer,x,y,text,fg,outline,outlineEnabled) { Array.from(text).forEach((ch,i)=>drawGlyph(buffer,x+i*4,y,ch,fg,outline,outlineEnabled)); }

/* Match the real menu OBJ artwork: 2,3,4,3,2 logical pixels wide. */
function drawMenuArrow(buffer,x,y,fg,outline,outlineEnabled) {
  const rowWidths=[2,3,4,3,2], points=[];
  for(let row=0;row<rowWidths.length;row++) for(let col=0;col<rowWidths[row];col++) points.push([x+col,y+row]);
  if(outlineEnabled) {
    for(const [px,py] of points) for(let oy=-1;oy<=1;oy++) for(let ox=-1;ox<=1;ox++) {
      const xx=px+ox,yy=py+oy;
      if(xx>=0&&yy>=0&&xx<FRAME_WIDTH&&yy<FRAME_HEIGHT) buffer[yy*FRAME_WIDTH+xx]=outline;
    }
  }
  for(const [px,py] of points) buffer[py*FRAME_WIDTH+px]=fg;
}

function menuPreviewScale(canvas) {
  const available=Math.max(FRAME_WIDTH,Math.floor(canvas.parentElement?.clientWidth||canvas.clientWidth||480));
  return Math.max(1,Math.min(4,Math.floor(available/FRAME_WIDTH)));
}

function paintIntegerScaled(canvas,logical) {
  const scale=menuPreviewScale(canvas),width=FRAME_WIDTH*scale,height=FRAME_HEIGHT*scale;
  if(canvas.width!==width) canvas.width=width;
  if(canvas.height!==height) canvas.height=height;
  canvas.style.width=`${width}px`;
  canvas.style.height=`${height}px`;
  const ctx=canvas.getContext('2d',{alpha:false});
  ctx.imageSmoothingEnabled=false;
  const image=ctx.createImageData(width,height);
  for(let y=0;y<FRAME_HEIGHT;y++) for(let x=0;x<FRAME_WIDTH;x++) {
    const [r,g,b]=rgb555ToRGB(logical[y*FRAME_WIDTH+x]);
    for(let sy=0;sy<scale;sy++) for(let sx=0;sx<scale;sx++) {
      const p=((y*scale+sy)*width+x*scale+sx)*4;
      image.data[p]=r; image.data[p+1]=g; image.data[p+2]=b; image.data[p+3]=255;
    }
  }
  ctx.putImageData(image,0,0);
}

function renderMenuPreview(canvas,theme,settings={},elapsed=0) {
  if(!canvas||!theme) return;
  const frameIndex=theme.kind==='frames'?Math.floor(elapsed/((theme.frameVBlanks||12)/59.7275*1000))%theme.frames.length:0;
  const palette=paletteForPreview(theme,elapsed),source=theme.frames[frameIndex]||theme.frames[0],logical=new Uint16Array(FRAME_BYTES);
  for(let i=0;i<FRAME_BYTES;i++) logical[i]=getColour(palette,source[i]);
  const colors=settingsColours(settings),outlineEnabled=!!settings.outline,ui=colors.ui,selected=colors.selected,outline=colors.outline;
  for(let x=0;x<FRAME_WIDTH;x++) {
    if(outlineEnabled) { logical[13*FRAME_WIDTH+x]=outline; logical[15*FRAME_WIDTH+x]=outline; }
    logical[14*FRAME_WIDTH+x]=ui;
  }
  drawText(logical,36,2,'SELECT MEDIA',ui,outline,outlineEnabled);
  drawText(logical,2,8,'CLIP 1/3',ui,outline,outlineEnabled);
  drawText(logical,74,8,'TOTAL 01:05',ui,outline,outlineEnabled);
  drawText(logical,8,17,'FIRST MEDIA',selected,outline,outlineEnabled);
  drawText(logical,8,23,'SECOND MEDIA',ui,outline,outlineEnabled);
  drawText(logical,8,29,'THIRD MEDIA',ui,outline,outlineEnabled);
  drawMenuArrow(logical,3,17,selected,outline,outlineEnabled);
  paintIntegerScaled(canvas,logical);
}
function startPreview(canvas,getTheme,getSettings) { let stopped=false,start=performance.now(); const tick=now=>{if(stopped)return;renderMenuPreview(canvas,getTheme(),getSettings(),now-start);requestAnimationFrame(tick);};requestAnimationFrame(tick);return()=>{stopped=true;}; }

const api={FRAME_WIDTH,FRAME_HEIGHT,FRAME_BYTES,RGB24_FRAME_BYTES,UI_PRESETS,OUTLINE_PRESETS,COMMON_GBA_COLORS,createBuiltinTheme,decodeCustomFile,decodeRGB24Frames,isVideoFile,serializeTheme,deserializeTheme,renderMenuPreview,startPreview,applyUI,settingsColours,rgb555ToHex,hexToRGB555,quantizeHexColor,describeColor,normalizeHexColor,setupGBAColorPicker};

global.MenuThemeTools=api;
})(window);
