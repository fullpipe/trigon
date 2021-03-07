# trigon


## TODO

- [ ] yaml config
    - [ ] input
    - [ ] triggers
        - [ ] pipes
        - [ ] output
- [ ] tests
- [ ] pipes
    - [x] log
    - [x] action
    - [x] every nth
    - [ ] debounce
    - [ ] expr filter, use https://github.com/antonmedv/expr for filtering


```yaml
input:
    host: 

triggers:
    - name: 'Log all'
      pipes: 
        - ['actionFilter', ['insert', 'update', 'delete']]
        - ['log']
        - ['webhook', {
            url: http://request.bin/asdasd
        }]

```
