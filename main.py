import math
from datetime import datetime

def pi(iteraciones):
    pi = 0.0
    for k in range(iteraciones):
        termino = (1 / (16 ** k)) * (
            4 / (8 * k + 1) -
            2 / (8 * k + 4) -
            1 / (8 * k + 5) -
            1 / (8 * k + 6)
        )
        pi += termino
    return pi

iteraciones = 10_000
pi_aproximado = pi(iteraciones)
print(pi_aproximado)
print(math.pi)
print(pi_aproximado == math.pi)
