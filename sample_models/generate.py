# generate a 2-class VW training set from the scikit-learn digits dataset.
# this script was used to generate 'digits.vw' in this directory.
import os
import pandas as pd
from sklearn.datasets import load_digits

model_file = "./digits.vw"
training_file = "./digits.train"

def generate_example(r):
    
    if r.label == 0:
        s = "-1 | "
    else:
        s = "1 | "
    
    for i in range(0, 64):
        s += "%d:%0.1f " % (i, r.loc[i])

    return s

d = load_digits(2)
df = pd.DataFrame(d.data)
df['label'] = d.target

exs = df.apply(generate_example, axis=1)
exs.to_csv(training_file, index=False)

os.system("vw --loss_function=logistic -d %s -f %s" % (training_file, model_file))
