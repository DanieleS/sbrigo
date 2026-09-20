-- Reparti predefiniti con la relativa legenda. Sono modificabili dall'interfaccia e dalle API.
INSERT INTO departments (id, name, description, default_sort_order) VALUES
    (gen_random_uuid(), 'Frutta e verdura', 'Prodotti freschi del banco ortofrutta', 10),
    (gen_random_uuid(), 'Dispensa',         'Pasta, riso, scatolame, conserve e tutto ciò che si conserva a temperatura ambiente', 20),
    (gen_random_uuid(), 'Frigo',            'Latticini, salumi, uova e prodotti da tenere in frigorifero', 30),
    (gen_random_uuid(), 'Surgelati',        'Prodotti del banco freezer', 40),
    (gen_random_uuid(), 'Casa',             'Detersivi, pulizia, carta e articoli non alimentari', 50),
    (gen_random_uuid(), 'Altro',            'Tutto ciò che non rientra negli altri reparti', 90);
