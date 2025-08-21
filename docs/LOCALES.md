# Locales
GhostRedact ships with international defaults only: email, phone, cc, ipv4, ipv6, iban.
## Brazil (optional)
Enable with:
```
ghostredact -in sample.txt -locale br
```
Adds: CPF, CNPJ, CEP, RG.
Or granular:
```
ghostredact -in sample.txt -types email,cc,cpf,cnpj,cep,rg
```
