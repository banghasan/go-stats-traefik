# Dokumentasi API

Layanan ini menyediakan API untuk memantau trafik yang melewati middleware
Traefik.

## Base URL

`/api/`

## Endpoints

### 0. Informasi Aplikasi

Mengambil informasi meta tentang aplikasi.

- **URL**: `GET /api` (tanpa slash di akhir)
- **Contoh**: `curl http://localhost:8080/api`

**Response**:

```json
{
  "app": "Go Stats Traefik",
  "version": "1.0.0",
  "build_info": "unknown"
}
```

### 1. Lihat Semua Host

Mengambil statistik dari **seluruh host** yang terekam.

- **URL**: `GET /api/`
- **Contoh**: `curl http://localhost:8080/api/`

**Response**:

```json
[
  {
    "host": "api.test.com",
    "total": 1500,
    "data": [ ... ]
  },
  {
    "host": "web.example.com",
    "total": 850,
    "data": [ ... ]
  }
]
```

> **Catatan**: Field `total` pada level host (atas) adalah total hit
> **keseluruhan** untuk host tersebut (tidak terpengaruh filter).

### 2. Lihat Detail Host

Mengambil statistik untuk **satu host spesifik**.

- **URL**: `GET /api/:host`
- **Contoh**: `curl http://localhost:8080/api/api.test.com`

### Parameter (Query Params)

Anda bisa menyaring data `data` (prefix path) menggunakan parameter berikut:

| Parameter | Contoh        | Deskripsi                                                                                         |
| --------- | ------------- | ------------------------------------------------------------------------------------------------- |
| `year`    | `?year=2025`  | Menampilkan data hanya untuk tahun tertentu. Default: semua tahun.                                |
| `prefix`  | `?prefix=/v1` | Menampilkan hanya path prefix tertentu.                                                           |
| `all`     | `?all=1`      | Jika `1` atau `true`, menampilkan **semua** path. Default: hanya menampilkan Top 20 path teratas. |

### Contoh Request Lengkap

**1. Tampilkan detail host `api.test.com` untuk tahun 2025 saja:**

```bash
curl "http://localhost:8080/api/api.test.com?year=2025"
```

**2. Tampilkan semua path (tanpa batas Top 20) untuk host tersebut:**

```bash
curl "http://localhost:8080/api/api.test.com?all=1"
```

**3. Cari spesifik prefix `/v3` pada host tersebut:**

```bash
curl "http://localhost:8080/api/api.test.com?prefix=/v3"
```

Response:

```json
[
  {
    "host": "api.test.com",
    "total": 5000,
    "data": [
      {
        "prefix": "/v3",
        "total": 120,
        "tahun": [2025, 2026]
      }
    ]
  }
]
```

_(Ingat: `total` 5000 adalah total global host, `total` 120 adalah total untuk
/v3)_

### Respon Kosong

Jika data tidak ditemukan, API akan mengembalikan array kosong:

```json
[]
```
