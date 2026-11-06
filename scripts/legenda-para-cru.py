"""Converte a legenda json3 do YouTube no texto `cru` de uma Referência.

Protótipo da validação manual (DECISOES §7, item 0); no app, isto vira Go.
Uso: python3 -I legenda-para-cru.py <legenda.json3> <info.json> <saida.md>
"""
import json
import re
import sys

BLOCO_MS = 30_000  # uma marca de tempo a cada ~30 s


def tempo(ms):
    s = ms // 1000
    h, m, s = s // 3600, s % 3600 // 60, s % 60
    return f"{h}:{m:02d}:{s:02d}" if h else f"{m:02d}:{s:02d}"


def main(legenda, info, saida):
    eventos = json.load(open(legenda, encoding="utf-8")).get("events", [])
    meta = json.load(open(info, encoding="utf-8"))

    blocos, atual, inicio = [], [], None
    for ev in eventos:
        if "segs" not in ev or ev.get("aAppend"):
            continue
        texto = "".join(s.get("utf8", "") for s in ev["segs"]).replace("\n", " ").strip()
        if not texto:
            continue
        t = ev.get("tStartMs", 0)
        if inicio is None:
            inicio = t
        elif t - inicio >= BLOCO_MS:
            blocos.append((inicio, " ".join(atual)))
            atual, inicio = [], t
        atual.append(texto)
    if atual:
        blocos.append((inicio, " ".join(atual)))

    if not blocos:
        sys.exit("legenda vazia")

    with open(saida, "w", encoding="utf-8") as f:
        f.write(f"# {meta['title']}\n\n")
        f.write(f"- Canal: {meta.get('channel')}\n- URL: {meta['webpage_url']}\n")
        f.write(f"- Duração: {meta.get('duration_string')}\n- Publicado: {meta.get('upload_date')}\n\n")
        for ini, texto in blocos:
            texto = re.sub(r"\s+", " ", texto)
            f.write(f"[{tempo(ini)}] {texto}\n\n")


if __name__ == "__main__":
    main(*sys.argv[1:4])
