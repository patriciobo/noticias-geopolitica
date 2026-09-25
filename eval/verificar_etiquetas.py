"""Verifica las etiquetas de eval/classify.jsonl con tres jueces independientes.

Tres modelos de proveedores distintos (Anthropic, Google, OpenAI) que NO
usa la plataforma para clasificar, así la evaluación no mide a los
modelos de producción contra sí mismos. Cada juez recibe el mismo
criterio que el clasificador (classifyCriteria en
core/internal/filter/claude.go) y decide por su cuenta.

Regla: una etiqueta se confirma o se cambia solo si los tres jueces
coinciden. Sin unanimidad, queda como estaba y se marca "sin consenso".
Además guarda la traducción al español de cada titular (title_es).

Uso (desde la raíz del repo, con OPENROUTER_API_KEY en el entorno o en
core/.env):

    python3 eval/verificar_etiquetas.py
"""

import json
import os
import re
import ssl
import sys
import time
import urllib.request

JUECES = ["anthropic/claude-sonnet-5", "google/gemini-3.5-flash", "openai/gpt-5.4-mini"]
LOTE = 20
RUTA = "eval/classify.jsonl"


def contexto_ssl():
    # Python instalado desde python.org en macOS no trae certificados: se
    # prueba certifi y el almacén del sistema antes del default.
    try:
        import certifi
        return ssl.create_default_context(cafile=certifi.where())
    except ImportError:
        pass
    if os.path.exists("/etc/ssl/cert.pem"):
        return ssl.create_default_context(cafile="/etc/ssl/cert.pem")
    return ssl.create_default_context()


SSL = contexto_ssl()


def clave():
    if os.environ.get("OPENROUTER_API_KEY"):
        return os.environ["OPENROUTER_API_KEY"]
    for linea in open("core/.env", encoding="utf-8"):
        if linea.startswith("OPENROUTER_API_KEY="):
            return linea.split("=", 1)[1].strip().strip("\"'")
    sys.exit("falta OPENROUTER_API_KEY")


def criterio():
    fuente = open("core/internal/filter/claude.go", encoding="utf-8").read()
    m = re.search(r"const classifyCriteria = `(.*?)`", fuente, re.S)
    return m.group(1)


def sistema():
    return (
        "Sos un editor que verifica etiquetas de un conjunto de evaluación. Cada titular "
        "viene de un medio de otro país y puede estar en cualquier idioma. Para cada uno: "
        "traducilo al español y decidí si es internacional según este criterio:\n\n"
        + criterio()
        + "\n\nSolo tenés el titular (no el copete). Respondé EXCLUSIVAMENTE con un array JSON, "
        'un objeto por titular: [{"id": "...", "title_es": "traducción al español", '
        '"is_international": true|false, "reason": "una oración"}]'
    )


def pedir(modelo, key, titulares):
    usuario = "\n".join(
        f'{t["id"]} | {t["source"]} ({t["country"]}): {t["title"]}' for t in titulares
    )
    cuerpo = json.dumps({
        "model": modelo,
        "messages": [
            {"role": "system", "content": sistema()},
            {"role": "user", "content": usuario},
        ],
        "temperature": 0,
        "max_tokens": 8000,
    }).encode()
    for intento in range(3):
        try:
            req = urllib.request.Request(
                "https://openrouter.ai/api/v1/chat/completions",
                data=cuerpo,
                headers={"Authorization": f"Bearer {key}", "Content-Type": "application/json"},
            )
            with urllib.request.urlopen(req, timeout=300, context=SSL) as r:
                texto = json.load(r)["choices"][0]["message"]["content"]
            texto = texto[texto.index("["): texto.rindex("]") + 1]
            return {d["id"]: d for d in json.loads(texto)}
        except Exception as e:  # noqa: BLE001 — se reintenta cualquier falla
            print(f"  {modelo}: intento {intento + 1} falló ({e})", file=sys.stderr)
            time.sleep(5 * (intento + 1))
    return {}


def main():
    key = clave()
    filas = [json.loads(l) for l in open(RUTA, encoding="utf-8") if l.strip()]
    votos = {f["id"]: {} for f in filas}
    traduccion = {}
    for modelo in JUECES:
        print(f"juez {modelo}...", file=sys.stderr)
        for i in range(0, len(filas), LOTE):
            for id_, d in pedir(modelo, key, filas[i:i + LOTE]).items():
                if id_ in votos and isinstance(d.get("is_international"), bool):
                    votos[id_][modelo] = d["is_international"]
                    traduccion.setdefault(id_, d.get("title_es", ""))

    # Si un juez no respondió nada, no hay tres votos para nadie: no se
    # toca el archivo.
    for modelo in JUECES:
        if not any(modelo in v for v in votos.values()):
            sys.exit(f"el juez {modelo} no respondió ningún lote: no se modifica {RUTA}")

    resumen = {"confirmadas": 0, "cambiadas": 0, "sin_consenso": 0}
    for f in filas:
        v = votos[f["id"]]
        f["title_es"] = traduccion.get(f["id"], f.get("title_es", ""))
        f["votes"] = v
        if len(v) == len(JUECES) and len(set(v.values())) == 1:
            unanime = next(iter(v.values()))
            if unanime == f["expected"]:
                f["verification"] = "confirmada por 3 jueces"
                resumen["confirmadas"] += 1
            else:
                f["verification"] = f"cambiada por 3 jueces (antes {str(f['expected']).lower()})"
                f["expected"] = unanime
                resumen["cambiadas"] += 1
            f["labeled_by"] = "consenso de 3 jueces (" + ", ".join(JUECES) + ")"
        else:
            f["verification"] = "sin consenso: se mantiene la etiqueta original"
            resumen["sin_consenso"] += 1

    with open(RUTA, "w", encoding="utf-8") as out:
        for f in filas:
            out.write(json.dumps(f, ensure_ascii=False) + "\n")
    print(json.dumps(resumen, ensure_ascii=False))


if __name__ == "__main__":
    main()
