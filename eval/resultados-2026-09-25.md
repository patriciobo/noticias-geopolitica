# Evaluación del clasificador

115 titulares etiquetados (../eval/classify.jsonl), 25 internacionales.

| Modelo | Precisión | Cobertura | F1 | Aciertos | Errores técnicos |
|---|---|---|---|---|---|
| `deepseek/deepseek-v4-flash` | 77% | 96% | 0.86 | 107/115 | 0 |
| `qwen/qwen3.8-27b:free` | 95% | 90% | 0.92 | 96/99 | 16 |
| `nvidia/nemotron-3-super-120b-a12b:free` | 67% | 90% | 0.77 | 72/83 | 32 |

## deepseek/deepseek-v4-flash — desacuerdos

- **Falso positivo** — Der Spiegel (Alemania): HVO: Ökodiesel im Großhandel erstmals billiger als Diesel
- **Falso positivo** — The Guardian Australia (Australia): Australia banned this anti-fascist play in 1936 at the request of the Nazis. Ninety years on, it’s being staged again
- **Falso positivo** — Mada Masr (Egipto): Court upholds sentence against Omnia Sweidan for publishing description of obstetric violence at Al-Shatby Hospital
- **Falso positivo** — Mada Masr (Egipto): Future of Egypt enters healthcare: Govt, military hospitals to serve tourist dollars
- **Falso positivo** — Le Parisien (Francia): Turquie-France : « On parle un peu plus du coach que de certains joueurs, c’est une bonne chose », remarque Adrien Rabiot _(deporte sin componente político)_
- **Falso positivo** — Novaya Gazeta Europe (Rusia): Красный Берлин
- **Falso positivo** — Mail & Guardian (Sudáfrica): The Quiet Majority for Human Rights
- **Falso negativo** — El Pitazo (Venezuela): Fiscalía rechaza libertad provisional de Cilia Flores por «extremo riesgo de fuga» _(proceso judicial en EE. UU. contra la esposa de Maduro: caso límite, depende del copete)_

## qwen/qwen3.8-27b:free — desacuerdos

- **Falso negativo** — Ciper Chile (Chile): Entre enero y agosto se detectaron 286 extranjeros cruzando ilegalmente la frontera norte: 93% menos que en igual periodo de 2025 _(migración entre países (Chile-Perú/Bolivia): cuenta por el criterio de migración)_
- **Falso positivo** — Fox News (Estados Unidos): American businesses aren’t allowed to build. It’s time Congress helped them
- **Falso negativo** — El Pitazo (Venezuela): Fiscalía rechaza libertad provisional de Cilia Flores por «extremo riesgo de fuga» _(proceso judicial en EE. UU. contra la esposa de Maduro: caso límite, depende del copete)_

## nvidia/nemotron-3-super-120b-a12b:free — desacuerdos

- **Falso positivo** — The Guardian Australia (Australia): Australia banned this anti-fascist play in 1936 at the request of the Nazis. Ninety years on, it’s being staged again
- **Falso positivo** — Mada Masr (Egipto): Court upholds sentence against Omnia Sweidan for publishing description of obstetric violence at Al-Shatby Hospital
- **Falso positivo** — Mada Masr (Egipto): Future of Egypt enters healthcare: Govt, military hospitals to serve tourist dollars
- **Falso positivo** — Le Parisien (Francia): Turquie-France : « On parle un peu plus du coach que de certains joueurs, c’est une bonne chose », remarque Adrien Rabiot _(deporte sin componente político)_
- **Falso negativo** — Libération (Francia): Le diesel au plus haut, la bataille entre Trump et les médias s’envenime, l’élection du RN à Montargis annulée… L’actu de ce jeudi
- **Falso positivo** — The Daily Telegraph (Reino Unido): Tax rises ‘virtually inevitable’ after £8bn borrowing blow
- **Falso positivo** — Novaya Gazeta Europe (Rusia): Красный Берлин
- **Falso positivo** — Novaya Gazeta Europe (Rusia): VK может незаметно включать функции сбора данных в «Макс» для отдельных пользователей
- **Falso positivo** — Meduza (Rusia): Russia ties family mortgage rates to the number of children. In Moscow, the rate for one-child families doubles to 12% on October 1.
- **Falso positivo** — The Moscow Times (Rusia): Firms Linked to Putin Classmate Win $1.5Bln in Moscow School Meal Contracts, Report Says
- **Falso negativo** — El Pitazo (Venezuela): Fiscalía rechaza libertad provisional de Cilia Flores por «extremo riesgo de fuga» _(proceso judicial en EE. UU. contra la esposa de Maduro: caso límite, depende del copete)_

