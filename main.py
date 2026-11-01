def hitung_matematika(a, b):
    if int(a) < 0 or int(b) < 0:
        raise ValueError("Input tidak boleh negatif")
    return (int(a) + b) / 2

def rata_rata(a, b):
    try:
        hasil = hitung_matematika(b, a)
        return hasil
    except TypeError:
        print("tidak bisa mengoperasikan data berbeda tipe")
        return -1
    except ValueError:
        print("Input harus angka positif")
        return -2

print(rata_rata('7', 3))









[].copy()
